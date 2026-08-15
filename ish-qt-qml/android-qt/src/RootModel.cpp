#include "RootModel.h"

#include <QDir>
#include <QFileInfo>
#include <QFileInfoList>

RootModel::RootModel(QObject *parent)
    : QAbstractListModel(parent)
{
}

void RootModel::setRootPath(const QString &path)
{
    const QString clean = QDir::cleanPath(path);
    if (m_rootPath == clean)
        return;
    m_rootPath = clean;
    emit rootPathChanged();
    refresh();
}

int RootModel::rowCount(const QModelIndex &parent) const
{
    return parent.isValid() ? 0 : m_entries.size();
}

QVariant RootModel::data(const QModelIndex &index, int role) const
{
    if (!index.isValid() || index.row() < 0 || index.row() >= m_entries.size())
        return {};
    const Entry &entry = m_entries.at(index.row());
    switch (role) {
    case Qt::DisplayRole:
    case NameRole:
        return entry.name;
    case PathRole:
        return entry.path;
    case DirectoryRole:
        return entry.directory;
    case SizeRole:
        return entry.size;
    default:
        return {};
    }
}

QHash<int, QByteArray> RootModel::roleNames() const
{
    return {
        {PathRole, QByteArrayLiteral("path")},
        {NameRole, QByteArrayLiteral("name")},
        {DirectoryRole, QByteArrayLiteral("directory")},
        {SizeRole, QByteArrayLiteral("size")}
    };
}

void RootModel::refresh()
{
    beginResetModel();
    m_entries.clear();
    if (!m_rootPath.isEmpty()) {
        const QDir dir(m_rootPath);
        const QFileInfoList infos = dir.entryInfoList(QDir::AllEntries | QDir::NoDotAndDotDot,
                                                      QDir::DirsFirst | QDir::Name);
        for (const QFileInfo &info : infos) {
            m_entries.append({info.absoluteFilePath(), info.fileName(), info.isDir(), info.size()});
        }
    }
    endResetModel();
}
