#pragma once

#include <QObject>
#include <QString>
#include <QList>

class RootfsManager final : public QObject
{
    Q_OBJECT
    Q_PROPERTY(bool prepared READ prepared NOTIFY preparedChanged)
    Q_PROPERTY(QString rootPath READ rootPath NOTIFY rootPathChanged)
    Q_PROPERTY(bool repositoryManaged READ repositoryManaged NOTIFY repositoryManagedChanged)
    Q_PROPERTY(bool repositoryUpdateRequired READ repositoryUpdateRequired NOTIFY repositoryUpdateRequiredChanged)
    Q_PROPERTY(int ishVersion READ ishVersion NOTIFY repositoryStateChanged)
    Q_PROPERTY(int apkVersion READ apkVersion NOTIFY repositoryStateChanged)
    Q_PROPERTY(QString currentApkVersion READ currentApkVersion CONSTANT)

public:
    explicit RootfsManager(QObject *parent = nullptr);

    bool prepared() const { return m_prepared; }
    QString rootPath() const { return m_rootPath; }
    bool repositoryManaged() const { return m_repositoryManaged; }
    bool repositoryUpdateRequired() const { return m_repositoryUpdateRequired; }
    int ishVersion() const { return m_ishVersion; }
    int apkVersion() const { return m_apkVersion; }
    QString currentApkVersion() const { return QStringLiteral("Alpine v3.19"); }

    Q_INVOKABLE void prepare();
    Q_INVOKABLE void resetInstalledData();
    Q_INVOKABLE void refreshRepositoryState();
    Q_INVOKABLE bool updateRepositories();
    Q_INVOKABLE QString terminalUrl(const QString &webChannelUrl,
                                    const QString &pageUrl = QString()) const;

signals:
    void preparedChanged();
    void rootPathChanged();
    void repositoryManagedChanged();
    void repositoryUpdateRequiredChanged();
    void repositoryStateChanged();
    void repositoriesUpdated();
    void progressChanged(int percent, const QString &message);
    void preparationError(const QString &message);

private:
    bool importBundledRootfs(const QString &archivePath, const QString &destination, QString *error);
    bool writeMetadata(const QString &destination, const QList<QByteArray> &paths,
                       const QList<QByteArray> &hardlinkTargets,
                       const QList<quint32> &modes, const QList<quint32> &uids,
                       const QList<quint32> &gids, QString *error);
    bool updateOnlyRepositoriesFile();
    bool writeTextFile(const QString &path, const QByteArray &contents);
    void initializeFilesystemMetadata();
    void setPrepared(bool value);

    static constexpr int kCurrentApkVersion = 31900;

    QString m_rootPath;
    bool m_prepared = false;
    bool m_repositoryManaged = false;
    bool m_repositoryUpdateRequired = false;
    int m_ishVersion = 0;
    int m_apkVersion = 0;
};
