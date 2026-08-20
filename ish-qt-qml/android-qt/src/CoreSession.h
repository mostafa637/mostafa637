#pragma once

#include <cstddef>

#include <QByteArray>
#include <QObject>
#include <QString>
#include <QStringList>

struct IshCoreSession;

class CoreSession final : public QObject
{
    Q_OBJECT

public:
    explicit CoreSession(QObject *parent = nullptr);
    ~CoreSession() override;

    bool start(const QString &rootPath,
               const QStringList &bootCommand,
               const QStringList &launchCommand);
    void stop();
    qint64 write(const QByteArray &bytes);
    void resize(int columns, int rows);
    bool isRunning() const { return m_running; }

signals:
    void outputReady(const QByteArray &bytes);
    void exited(int exitCode);
    void errorOccurred(const QString &message);
    void runningChanged(bool running);

private:
    static void outputCallback(void *cookie, const char *bytes, size_t length);
    static void stateCallback(void *cookie, int exitCode);

    static QByteArray hostResolverConfig();
    static QByteArray resolverConfigFromFile(const QString &path);
#if defined(Q_OS_ANDROID)
    static QByteArray resolverConfigFromAndroidConnectivity();
#endif

    void handleOutput(const QByteArray &bytes);
    void handleState(int exitCode);
    void setRunning(bool running);
    void destroyCore();

    IshCoreSession *m_session = nullptr;
    bool m_running = false;
    QByteArray m_pendingOutputLine;
    QByteArray m_probeOutput;
    bool m_pythonProbeReported = false;
};
