#include "CoreSession.h"

#include <QFile>
#include <QMetaObject>
#include <QPointer>
#include <QRegularExpression>
#include <QThread>

#include <vector>

#if defined(Q_OS_ANDROID)
#include <QCoreApplication>
#include <QJniObject>
#endif

extern "C" {
#include "../../upstream/ish-ios/app/core/CoreSession.h"
}

namespace {

QByteArray filterResolverConfig(const QByteArray &input)
{
    QByteArray result;
    const QList<QByteArray> lines = input.split('\n');
    for (QByteArray line : lines) {
        line = line.trimmed();
        if (line.isEmpty() || line.startsWith('#'))
            continue;
        const int comment = line.indexOf('#');
        if (comment >= 0)
            line.truncate(comment);
        line = line.trimmed();
        if (line.startsWith("nameserver ") || line.startsWith("search ")) {
            result += line;
            result += '\n';
        }
    }
    return result;
}

} // namespace

CoreSession::CoreSession(QObject *parent)
    : QObject(parent)
{
}

CoreSession::~CoreSession()
{
    stop();
    destroyCore();
}

QByteArray CoreSession::resolverConfigFromFile(const QString &path)
{
    QFile file(path);
    if (!file.open(QIODevice::ReadOnly | QIODevice::Text))
        return {};
    return filterResolverConfig(file.readAll());
}

#if defined(Q_OS_ANDROID)
QByteArray CoreSession::resolverConfigFromAndroidConnectivity()
{
    const QJniObject context = QNativeInterface::QAndroidApplication::context();
    if (!context.isValid())
        return {};

    const QJniObject serviceName = QJniObject::fromString(QStringLiteral("connectivity"));
    const QJniObject connectivity = context.callObjectMethod(
        "getSystemService", "(Ljava/lang/String;)Ljava/lang/Object;", serviceName.object());
    if (!connectivity.isValid())
        return {};

    const QJniObject network = connectivity.callObjectMethod(
        "getActiveNetwork", "()Landroid/net/Network;");
    if (!network.isValid())
        return {};

    const QJniObject linkProperties = connectivity.callObjectMethod(
        "getLinkProperties", "(Landroid/net/Network;)Landroid/net/LinkProperties;",
        network.object());
    if (!linkProperties.isValid())
        return {};

    const QJniObject dnsServers = linkProperties.callObjectMethod(
        "getDnsServers", "()Ljava/util/List;");
    if (!dnsServers.isValid())
        return {};

    QByteArray result;
    const jint count = dnsServers.callMethod<jint>("size", "()I");
    for (jint i = 0; i < count; ++i) {
        const QJniObject address = dnsServers.callObjectMethod(
            "get", "(I)Ljava/lang/Object;", i);
        if (!address.isValid())
            continue;
        const QJniObject hostAddress = address.callObjectMethod(
            "getHostAddress", "()Ljava/lang/String;");
        const QString value = hostAddress.toString().trimmed();
        if (!value.isEmpty())
            result += "nameserver " + value.toUtf8() + '\n';
    }
    return result;
}
#endif

QByteArray CoreSession::hostResolverConfig()
{
    QByteArray result;
#if defined(Q_OS_ANDROID)
    result = resolverConfigFromAndroidConnectivity();
    if (!result.isEmpty())
        return result;
#endif

    result = resolverConfigFromFile(QStringLiteral("/etc/resolv.conf"));
    if (result.isEmpty())
        result = resolverConfigFromFile(QStringLiteral("/system/etc/resolv.conf"));
    return result;
}

bool CoreSession::start(const QString &rootPath,
                        const QStringList &bootCommand,
                        const QStringList &launchCommand)
{
    qWarning() << "[ish-qt] CoreSession::start() rootPath=" << rootPath
               << "boot=" << bootCommand << "launch=" << launchCommand;
    if (m_session != nullptr)
        stop();
    destroyCore();
    m_pendingOutputLine.clear();
    m_probeOutput.clear();
    m_pythonProbeReported = false;

    if (rootPath.isEmpty()) {
        qWarning() << "[ish-qt] CoreSession::start FAILED: rootfs path is empty";
        emit errorOccurred(QStringLiteral("Rootfs path is empty"));
        return false;
    }

    const QByteArray rootBytes = rootPath.toUtf8();
    QList<QByteArray> bootBytes;
    QList<QByteArray> launchBytes;
    std::vector<const char *> bootArgv;
    std::vector<const char *> launchArgv;
    bootBytes.reserve(bootCommand.size());
    launchBytes.reserve(launchCommand.size());
    bootArgv.reserve(bootCommand.size());
    launchArgv.reserve(launchCommand.size());

    for (const QString &value : bootCommand)
        bootBytes.append(value.toUtf8());

    // iSH iOS starts /bin/login on a controlling pseudo-terminal. Retain
    // compatibility with the old saved default without launching login.
    const QStringList legacyLogin = {
        QStringLiteral("/bin/login"), QStringLiteral("-f"), QStringLiteral("root")
    };
    QStringList effectiveLaunchCommand = launchCommand == legacyLogin
        ? QStringList{QStringLiteral("/bin/sh")}
        : launchCommand;

    // With a real PTY, /bin/sh detects an interactive terminal itself. Do
    // not force -i: this preserves the shell's normal tty/job-control mode
    // and avoids changing command semantics for explicit launch commands.
    if (effectiveLaunchCommand.isEmpty())
        effectiveLaunchCommand = {QStringLiteral("/bin/sh")};
    for (const QString &value : effectiveLaunchCommand)
        launchBytes.append(value.toUtf8());
    for (const QByteArray &value : bootBytes)
        bootArgv.push_back(value.constData());
    for (const QByteArray &value : launchBytes)
        launchArgv.push_back(value.constData());

    m_session = ish_core_session_create(
        rootBytes.constData(),
        bootArgv.empty() ? nullptr : bootArgv.data(), bootArgv.size(),
        launchArgv.empty() ? nullptr : launchArgv.data(), launchArgv.size(),
        &CoreSession::outputCallback, &CoreSession::stateCallback, this);
    if (!m_session) {
        qWarning() << "[ish-qt] CoreSession::start FAILED: ish_core_session_create returned null";
        emit errorOccurred(QStringLiteral("Unable to allocate iSH core session"));
        return false;
    }
    qWarning() << "[ish-qt] ish_core_session_create OK";

    const QByteArray resolverConfig = hostResolverConfig();
    qWarning() << "[ish-qt] hostResolverConfig size=" << resolverConfig.size();
    if (!resolverConfig.isEmpty() &&
        ish_core_session_set_resolver_config(m_session,
                                             resolverConfig.constData(),
                                             static_cast<size_t>(resolverConfig.size())) !=
            static_cast<size_t>(resolverConfig.size())) {
        emit errorOccurred(QStringLiteral("Unable to configure host DNS for iSH"));
        destroyCore();
        return false;
    }

    if (!ish_core_session_start(m_session)) {
        qWarning() << "[ish-qt] CoreSession::start FAILED: ish_core_session_start returned false";
        emit errorOccurred(QStringLiteral("Unable to start iSH core session"));
        destroyCore();
        return false;
    }
    setRunning(true);
    qWarning() << "[ish-qt] CoreSession::start SUCCEEDED, running=" << m_running;
    return true;
}

void CoreSession::stop()
{
    // Closing the PTY master delivers hangup/EIO to the shell and unblocks
    // the reader, allowing the fakefs mounts and sqlite metadata database
    // to close cleanly.
    if (m_session != nullptr) {
        struct IshCoreSession *session = m_session;
        ish_core_session_stop(session);
#if defined(__ANDROID__)
        (void)session;
#else
        (void)session;
#endif
    }
    setRunning(false);
}

qint64 CoreSession::write(const QByteArray &bytes)
{
    if (m_session == nullptr || bytes.isEmpty() || !m_running)
        return 0;

    // Pass terminal input through unchanged. The PTY's termios line
    // discipline handles carriage return, erase, flow control, and signals.
    return static_cast<qint64>(ish_core_session_write(
        m_session, bytes.constData(), static_cast<size_t>(bytes.size())));
}

void CoreSession::resize(int columns, int rows)
{
    if (m_session != nullptr && m_running)
        ish_core_session_resize(m_session, columns, rows);
}

void CoreSession::outputCallback(void *cookie, const char *bytes, size_t length)
{
    auto *self = static_cast<CoreSession *>(cookie);
    if (self == nullptr || bytes == nullptr || length == 0)
        return;
    const QPointer<CoreSession> guard(self);
    const QByteArray copy(bytes, static_cast<qsizetype>(length));
    QMetaObject::invokeMethod(self, [guard, copy]() {
        if (guard != nullptr)
            guard->handleOutput(copy);
    }, Qt::QueuedConnection);
}

void CoreSession::stateCallback(void *cookie, int exitCode)
{
    auto *self = static_cast<CoreSession *>(cookie);
    if (self == nullptr)
        return;
    const QPointer<CoreSession> guard(self);
    QMetaObject::invokeMethod(self, [guard, exitCode]() {
        if (guard != nullptr)
            guard->handleState(exitCode);
    }, Qt::QueuedConnection);
}

void CoreSession::handleOutput(const QByteArray &bytes)
{
    if (bytes.isEmpty())
        return;
    // The AVD smoke test sends `python3 --version` after installing the
    // package.  Output is delivered in arbitrary chunks, so retain a small
    // bounded probe buffer and emit a log marker only after the shell has
    // actually produced a Python version (or an explicit not-found result).
    if (!m_pythonProbeReported) {
        m_probeOutput += bytes;
        if (m_probeOutput.size() > 8192)
            m_probeOutput.remove(0, m_probeOutput.size() - 8192);
        if (m_probeOutput.contains("Python 3.")) {
            m_pythonProbeReported = true;
        } else if (m_probeOutput.contains("python3: not found") ||
                   m_probeOutput.contains("python3: not found\\r")) {
            m_pythonProbeReported = true;
        }
    }

    // Chromium/QtWebEngine prints repetitive diagnostic lines to the process
    // stderr (e.g. `[mmdd/hhmmss.xxx:ERROR:third_party/crashpad/...]` or GLES
    // shader-binding lines). CoreSession captures stderr together with the
    // shell stream, so drop these known Chromium/GLES diagnostic lines before
    // forwarding output to the terminal. All user/program output is preserved.
    static const char *const kDroppedPrefixes[] = {
        "s_glBindAttribLocation:",
        "third_party/crashpad/",
        "third_party/blink/",
        "ERROR:crashpad",
        "ERROR:gpu",
        "INFO:crashpad"
    };
    static const QRegularExpression crashpadRe(
        QStringLiteral("^\\[\\d\\d\\d\\d/\\d\\d\\d\\d\\d\\d\\.\\d+:"),
        QRegularExpression::DotMatchesEverythingOption);
    auto isDiagnostic = [](const QByteArray &line) {
        if (line.isEmpty())
            return false;
        for (const char *prefix : kDroppedPrefixes) {
            if (line.startsWith(prefix))
                return true;
        }
        return crashpadRe.match(line).hasMatch();
    };
    m_pendingOutputLine += bytes;
    QByteArray forwarded;
    qsizetype consumed = 0;
    while (true) {
        const qsizetype newline = m_pendingOutputLine.indexOf('\n', consumed);
        if (newline < 0)
            break;
        const QByteArray line = m_pendingOutputLine.mid(consumed, newline - consumed);
        if (!isDiagnostic(line)) {
            forwarded += line;
            forwarded += '\n';
        }
        consumed = newline + 1;
    }
    if (consumed > 0)
        m_pendingOutputLine.remove(0, consumed);

    // Shell prompts and other terminal output are commonly not terminated by
    // a newline. Forward a normal partial line immediately so the WebView can
    // display the prompt and accept input. Keep only a partial known GLES
    // diagnostic line until its newline arrives, otherwise it could leak into
    // the terminal while being assembled across callbacks.
    if (!m_pendingOutputLine.isEmpty() && !isDiagnostic(m_pendingOutputLine)) {
        forwarded += m_pendingOutputLine;
        m_pendingOutputLine.clear();
    }

    // Avoid unbounded growth if a diagnostic producer never emits a newline.
    if (m_pendingOutputLine.size() > 64 * 1024) {
        m_pendingOutputLine.clear();
    }
    if (!forwarded.isEmpty())
        emit outputReady(forwarded);
}

void CoreSession::handleState(int exitCode)
{
    setRunning(false);
    emit exited(exitCode);
}

void CoreSession::setRunning(bool running)
{
    if (m_running == running)
        return;
    m_running = running;
    emit runningChanged(m_running);
}

void CoreSession::destroyCore()
{
    if (m_session == nullptr)
        return;
    ish_core_session_destroy(m_session);
    m_session = nullptr;
}
