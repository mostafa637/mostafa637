#include "CoreSession.h"

#include <QFile>
#include <QMetaObject>
#include <QPointer>
#include <QRegularExpression>
#include <QThread>

#include <vector>

#if defined(Q_OS_ANDROID)
#include <QJniObject>
#include <QtCore/qnativeinterface.h>
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
    if (m_session != nullptr)
        stop();
    destroyCore();

    if (rootPath.isEmpty()) {
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
    for (const QString &value : launchCommand)
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
        emit errorOccurred(QStringLiteral("Unable to allocate iSH core session"));
        return false;
    }

    const QByteArray resolverConfig = hostResolverConfig();
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
        emit errorOccurred(QStringLiteral("Unable to start iSH core session"));
        destroyCore();
        return false;
    }
    setRunning(true);
    return true;
}

void CoreSession::stop()
{
    if (m_session != nullptr)
        ish_core_session_stop(m_session);
    setRunning(false);
}

qint64 CoreSession::write(const QByteArray &bytes)
{
    if (m_session == nullptr || bytes.isEmpty() || !m_running)
        return 0;
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
    if (!bytes.isEmpty())
        emit outputReady(bytes);
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
