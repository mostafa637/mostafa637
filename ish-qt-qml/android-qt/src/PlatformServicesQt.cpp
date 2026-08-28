#include "PlatformServicesQt.h"

#include <QClipboard>
#include <QDateTime>
#include <QDir>
#include <QFile>
#include <QGuiApplication>
#include <QStandardPaths>
#include <QTextStream>

#include <csignal>
#include <cstring>
#include <fcntl.h>
#include <unistd.h>

namespace {
QByteArray g_crashLogPath;
QtMessageHandler g_previousMessageHandler = nullptr;

void writeCrashRecord(const char *record)
{
    if (g_crashLogPath.isEmpty())
        return;
    const int fd = ::open(g_crashLogPath.constData(), O_WRONLY | O_CREAT | O_APPEND, 0600);
    if (fd < 0)
        return;
    const size_t length = std::strlen(record);
    const ssize_t written = ::write(fd, record, length);
    (void)written;
    ::close(fd);
}

void crashSignalHandler(int signal)
{
    switch (signal) {
    case SIGABRT: writeCrashRecord("fatal signal: SIGABRT\\n"); break;
    case SIGBUS: writeCrashRecord("fatal signal: SIGBUS\\n"); break;
    case SIGFPE: writeCrashRecord("fatal signal: SIGFPE\\n"); break;
    case SIGILL: writeCrashRecord("fatal signal: SIGILL\\n"); break;
    case SIGSEGV: writeCrashRecord("fatal signal: SIGSEGV\\n"); break;
    default: writeCrashRecord("fatal signal: unknown\\n"); break;
    }
    std::signal(signal, SIG_DFL);
    std::raise(signal);
}

void qtMessageHandler(QtMsgType type, const QMessageLogContext &context, const QString &message)
{
    Q_UNUSED(context);
    const char *kind = "debug";
    switch (type) {
    case QtInfoMsg: kind = "info"; break;
    case QtWarningMsg: kind = "warning"; break;
    case QtCriticalMsg: kind = "critical"; break;
    case QtFatalMsg: kind = "fatal"; break;
    case QtDebugMsg: break;
    }
    const QByteArray record = QByteArray("qt ") + kind + ": " + message.toUtf8() + '\n';
    writeCrashRecord(record.constData());
    if (g_previousMessageHandler)
        g_previousMessageHandler(type, context, message);
}
} // namespace

PlatformServicesQt::PlatformServicesQt(QObject *parent)
    : QObject(parent)
{
    QStringList candidates;
#if defined(Q_OS_ANDROID)
    // The app-specific external directory is writable without a storage
    // permission and remains visible under /sdcard. Try public Download too
    // so older Android versions and adb users can access a shorter path.
    candidates << QStringLiteral("/sdcard/Android/data/com.mostafa637.ishqt/files/ish-qt-errors.log");
    candidates << QStringLiteral("/sdcard/Download/ish-qt-errors.log");
#endif
    const QString appData = QStandardPaths::writableLocation(QStandardPaths::AppDataLocation);
    if (!appData.isEmpty())
        candidates << QDir(appData).filePath(QStringLiteral("ish-qt-errors.log"));
    candidates << QDir(QDir::tempPath()).filePath(QStringLiteral("ish-qt-errors.log"));

    for (const QString &candidate : candidates) {
        const QFileInfo info(candidate);
        QDir().mkpath(info.absolutePath());
        auto *file = new QFile(candidate, this);
        if (file->open(QIODevice::WriteOnly | QIODevice::Append | QIODevice::Text)) {
            m_diagnosticLog = file;
            m_diagnosticLogPath = candidate;
            appendDiagnostic(QStringLiteral("diagnostic log opened"));
            break;
        }
        delete file;
    }

    if (QGuiApplication::clipboard()) {
        connect(QGuiApplication::clipboard(), &QClipboard::dataChanged,
                this, &PlatformServicesQt::clipboardChanged);
    }
}

QString PlatformServicesQt::pasteText() const
{
    return clipboardText();
}

QString PlatformServicesQt::clipboardText() const
{
    const QClipboard *clipboard = QGuiApplication::clipboard();
    if (!clipboard)
        return {};
    return clipboard->text(QClipboard::Clipboard);
}

void PlatformServicesQt::copyText(const QString &text)
{
    QClipboard *clipboard = QGuiApplication::clipboard();
    if (!clipboard)
        return;
    clipboard->setText(text, QClipboard::Clipboard);
}

void PlatformServicesQt::clearClipboard()
{
    QClipboard *clipboard = QGuiApplication::clipboard();
    if (!clipboard)
        return;
    clipboard->clear(QClipboard::Clipboard);
}

bool PlatformServicesQt::hasClipboardText() const
{
    return !clipboardText().isEmpty();
}

void PlatformServicesQt::reportError(const QString &title, const QString &message)
{
    logDiagnostic(title, message);
    emit errorReported(title, message);
}

void PlatformServicesQt::logDiagnostic(const QString &title, const QString &message)
{
    const QString safeTitle = title.simplified();
    const QString safeMessage = message;
    appendDiagnostic(QStringLiteral("%1: %2").arg(safeTitle, safeMessage));
}

void PlatformServicesQt::installCrashHandler()
{
    if (m_diagnosticLogPath.isEmpty())
        return;
    g_crashLogPath = m_diagnosticLogPath.toUtf8();
    g_previousMessageHandler = qInstallMessageHandler(qtMessageHandler);
    std::signal(SIGABRT, crashSignalHandler);
    std::signal(SIGBUS, crashSignalHandler);
    std::signal(SIGFPE, crashSignalHandler);
    std::signal(SIGILL, crashSignalHandler);
    std::signal(SIGSEGV, crashSignalHandler);
    appendDiagnostic(QStringLiteral("crash and Qt message handlers installed"));
}

void PlatformServicesQt::appendDiagnostic(const QString &line)
{
    if (!m_diagnosticLog || !m_diagnosticLog->isOpen())
        return;

    const QString timestamp = QDateTime::currentDateTimeUtc().toString(Qt::ISODateWithMs);
    const QString record = QStringLiteral("[%1] %2\n").arg(timestamp, line);
    m_diagnosticLog->write(record.toUtf8());
    m_diagnosticLog->flush();
}
