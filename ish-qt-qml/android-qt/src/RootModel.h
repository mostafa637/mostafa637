#pragma once

#include <QAbstractListModel>
#include <QString>

struct fakefsify_error;

class RootModel : public QAbstractListModel
{
    Q_OBJECT
    Q_PROPERTY(QString rootPath READ rootPath WRITE setRootPath NOTIFY rootPathChanged)
    Q_PROPERTY(QString defaultRoot READ defaultRoot WRITE setDefaultRoot NOTIFY defaultRootChanged)

public:
    enum Roles {
        PathRole = Qt::UserRole + 1,
        NameRole,
        DirectoryRole,
        SizeRole
    };
    Q_ENUM(Roles)

    explicit RootModel(QObject *parent = nullptr, bool managesRoots = true);

    QString rootPath() const { return m_rootPath; }
    void setRootPath(const QString &path);
    QString defaultRoot() const { return m_defaultRoot; }
    void setDefaultRoot(const QString &name);

    int rowCount(const QModelIndex &parent = QModelIndex()) const override;
    QVariant data(const QModelIndex &index, int role = Qt::DisplayRole) const override;
    QHash<int, QByteArray> roleNames() const override;

    Q_INVOKABLE void refresh();
    Q_INVOKABLE bool importArchive(const QString &archivePath, const QString &name);
    Q_INVOKABLE bool exportArchive(const QString &name, const QString &archivePath);
    Q_INVOKABLE bool destroyRoot(const QString &name);
    Q_INVOKABLE bool renameRoot(const QString &name, const QString &newName);
    Q_INVOKABLE bool isRootNameValid(const QString &name) const;

signals:
    void rootPathChanged();
    void defaultRootChanged();
    void operationProgress(int percent, const QString &message);
    void operationError(const QString &message);
    void rootsChanged();

protected:
    struct Entry {
        QString path;
        QString name;
        bool directory = false;
        qint64 size = 0;
    };
    QList<Entry> m_entries;

private:
    static void progressCallback(void *cookie, double progress, const char *message, bool *shouldCancel);
    bool ensureRootDirectory(QString *error) const;
    QString rootDirectoryForName(const QString &name) const;
    bool reportFakefsError(const QString &operation, const struct fakefsify_error &error);

    QString m_rootPath;
    QString m_defaultRoot;
    bool m_managesRoots = true;
};
