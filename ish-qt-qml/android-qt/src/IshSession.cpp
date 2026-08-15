#include "IshSession.h"

#include <QMetaObject>
#include <QThread>

#include <vector>

extern "C" {
#include "CoreSession.h"
}

IshSession::IshSession(QObject *parent)
    : QObject(parent)
{
    m_currentStyle.insert(QStringLiteral("fontFamily"), QStringLiteral("Noto Sans Mono"));
    m_currentStyle.insert(QStringLiteral("fontSize"), 14);
    m_currentStyle.insert(QStringLiteral("foregroundColor"), QStringLiteral("#f5f5f7"));
    m_currentStyle.insert(QStringLiteral("backgroundColor"), QStringLiteral("#000000"));
    m_currentStyle.insert(QStringLiteral("selectionColor"), QStringLiteral("#3a78d4"));
    m_currentStyle.insert(QStringLiteral("selectedTextColor"), QStringLiteral("#ffffff"));
    m_currentStyle.insert(QStringLiteral("blinkCursor"), true);
    m_currentStyle.insert(QStringLiteral("cursorShape"), QStringLiteral("block"));
}

IshSession::~IshSession()
{
    stop();
    destroyCore();
}

void IshSession::configure(const QString &rootPath,
                           const QStringList &bootCommand,
                           const QStringList &launchCommand)
{
    if (m_alive)
        stop();
    destroyCore();
    m_rootPath = rootPath;
    m_bootCommand = bootCommand;
    m_launchCommand = launchCommand;
}

bool IshSession::start(const QString &rootPath,
                       const QStringList &bootCommand,
                       const QStringList &launchCommand)
{
    configure(rootPath, bootCommand, launchCommand);
    if (m_rootPath.isEmpty()) {
        emit sessionError(QStringLiteral("Rootfs path is empty"));
        return false;
    }

    const QByteArray rootBytes = m_rootPath.toUtf8();
    QList<QByteArray> bootBytes;
    QList<QByteArray> launchBytes;
    std::vector<const char *> bootArgv;
    std::vector<const char *> launchArgv;

    for (const QString &value : m_bootCommand)
        bootBytes.append(value.toUtf8());
    for (const QString &value : m_launchCommand)
        launchBytes.append(value.toUtf8());
    for (const QByteArray &value : bootBytes)
        bootArgv.push_back(value.constData());
    for (const QByteArray &value : launchBytes)
        launchArgv.push_back(value.constData());

    m_core = ish_core_session_create(
        rootBytes.constData(),
        bootArgv.empty() ? nullptr : bootArgv.data(), bootArgv.size(),
        launchArgv.empty() ? nullptr : launchArgv.data(), launchArgv.size(),
        &IshSession::outputCallback, &IshSession::stateCallback, this);
    if (!m_core) {
        emit sessionError(QStringLiteral("Unable to allocate iSH core session"));
        return false;
    }

    m_stopRequested = false;
    if (!ish_core_session_start(m_core)) {
        emit sessionError(QStringLiteral("Unable to start iSH core session"));
        destroyCore();
        return false;
    }
    setAlive(true);
    return true;
}

void IshSession::stop()
{
    if (!m_core) {
        setAlive(false);
        return;
    }
    m_stopRequested = true;
    ish_core_session_stop(m_core);
    setAlive(false);
}

void IshSession::load()
{
    emit styleChanged(m_currentStyle);
    emit loaded();
}

void IshSession::sendInput(const QString &value)
{
    if (!m_core || !m_alive || value.isEmpty())
        return;
    const QByteArray bytes = value.toUtf8();
    ish_core_session_write(m_core, bytes.constData(), static_cast<size_t>(bytes.size()));
}

void IshSession::resize(int columns, int rows)
{
    if (m_core && m_alive)
        ish_core_session_resize(m_core, columns, rows);
}

void IshSession::controlModifierConsumed()
{
    emit controlModifierConsumedSignal();
}

void IshSession::setStyle(const QVariantMap &style)
{
    if (style == m_currentStyle)
        return;
    m_currentStyle = style;
    emit styleChanged(m_currentStyle);
}

void IshSession::outputCallback(void *cookie, const char *bytes, size_t length)
{
    auto *self = static_cast<IshSession *>(cookie);
    if (!self || !bytes || length == 0)
        return;
    const QByteArray copy(bytes, static_cast<qsizetype>(length));
    QMetaObject::invokeMethod(self, [self, copy]() { self->handleOutput(copy); }, Qt::QueuedConnection);
}

void IshSession::stateCallback(void *cookie, int exitCode)
{
    auto *self = static_cast<IshSession *>(cookie);
    if (!self)
        return;
    QMetaObject::invokeMethod(self, [self, exitCode]() { self->handleState(exitCode); }, Qt::QueuedConnection);
}

void IshSession::handleOutput(const QByteArray &bytes)
{
    if (!bytes.isEmpty())
        emit outputReady(QString::fromUtf8(bytes));
}

void IshSession::handleState(int exitCode)
{
    setAlive(false);
    if (!m_stopRequested && exitCode != 0)
        emit sessionError(QStringLiteral("iSH session ended with status %1").arg(exitCode));
}

void IshSession::setAlive(bool value)
{
    if (m_alive == value)
        return;
    m_alive = value;
    emit aliveChanged(m_alive);
}

void IshSession::destroyCore()
{
    if (!m_core)
        return;
    ish_core_session_destroy(m_core);
    m_core = nullptr;
}
