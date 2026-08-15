#pragma once

#include <QAbstractListModel>
#include <QString>

class RootModel : public QAbstractListModel
{
    Q_OBJECT
    Q_PROPERTY(QString rootPath READ rootPath WRITE setRootPath NOTIFY rootPathChanged)

public:
    enum Roles {
        PathRole = Qt::UserRole + 1,
        NameRole,
        DirectoryRole,
        SizeRole
    };
    Q_ENUM(Roles)

    explicit RootModel(QObject *parent = nullptr);

    QString rootPath() const { return m_rootPath; }
    void setRootPath(const QString &path);

    int rowCount(const QModelIndex &parent = QModelIndex()) const override;
    QVariant data(const QModelIndex &index, int role = Qt::DisplayRole) const override;
    QHash<int, QByteArray> roleNames() const override;

    Q_INVOKABLE void refresh();

signals:
    void rootPathChanged();

protected:
    struct Entry {
        QString path;
        QString name;
        bool directory = false;
        qint64 size = 0;
    };
    QList<Entry> m_entries;

private:
    QString m_rootPath;
};
