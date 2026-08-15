#pragma once

#include <QObject>
#include <QString>

class RootfsManager;

class RootfsUpgradeController final : public QObject
{
    Q_OBJECT
    Q_PROPERTY(bool busy READ busy NOTIFY busyChanged)
    Q_PROPERTY(int progress READ progress NOTIFY progressChanged)
    Q_PROPERTY(QString message READ message NOTIFY messageChanged)

public:
    explicit RootfsUpgradeController(RootfsManager *manager, QObject *parent = nullptr);

    bool busy() const { return m_busy; }
    int progress() const { return m_progress; }
    QString message() const { return m_message; }

    Q_INVOKABLE void reinstallBundledRootfs();

signals:
    void busyChanged();
    void progressChanged();
    void messageChanged();
    void failed(const QString &message);

private slots:
    void onManagerProgress(int percent, const QString &message);
    void onManagerError(const QString &message);
    void finishIfPrepared();

private:
    void setBusy(bool value);
    void setProgress(int value);
    void setMessage(const QString &value);

    RootfsManager *m_manager = nullptr;
    bool m_busy = false;
    int m_progress = 0;
    QString m_message;
};
