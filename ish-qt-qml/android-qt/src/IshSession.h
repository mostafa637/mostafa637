#pragma once

#include <QObject>
#include <QVariantMap>
#include <QStringList>

#include "CoreSession.h"

class IshSession final : public QObject
{
    Q_OBJECT
    Q_PROPERTY(bool alive READ alive NOTIFY aliveChanged)
    Q_PROPERTY(QVariantMap currentStyle READ currentStyle NOTIFY styleChanged)

public:
    explicit IshSession(QObject *parent = nullptr);
    ~IshSession() override;

    bool alive() const { return m_alive; }
    QVariantMap currentStyle() const { return m_currentStyle; }

    Q_INVOKABLE void configure(const QString &rootPath,
                               const QStringList &bootCommand,
                               const QStringList &launchCommand);
    Q_INVOKABLE bool start(const QString &rootPath,
                           const QStringList &bootCommand,
                           const QStringList &launchCommand);
    Q_INVOKABLE void stop();
    Q_INVOKABLE void load();
    Q_INVOKABLE void sendInput(const QString &value);
    Q_INVOKABLE void resize(int columns, int rows);
    Q_INVOKABLE void controlModifierConsumed();

public slots:
    void setStyle(const QVariantMap &style);

signals:
    void aliveChanged(bool alive);
    void outputReady(const QString &value);
    void styleChanged(const QVariantMap &style);
    void loaded();
    void sessionError(const QString &message);
    void controlModifierConsumedSignal();

private slots:
    void handleOutput(const QByteArray &bytes);
    void handleState(int exitCode);
    void handleCoreError(const QString &message);

private:
    void setAlive(bool value);

    CoreSession *m_core = nullptr;
    QString m_rootPath;
    QStringList m_bootCommand;
    QStringList m_launchCommand;
    QVariantMap m_currentStyle;
    bool m_alive = false;
    bool m_stopRequested = false;
};
