#include "RootModel.h"

extern "C" {
#include "tools/fakefs.h"
}

#include <QDateTime>
#include <QDir>
#include <QFileInfo>
#include <QFileInfoList>
#include <QSettings>
#include <QStandardPaths>

#include <cstdlib>

namespace {

constexpr auto kDefaultRootKey = "defaultRoot";

QString fakefsErrorText(const struct fakefsify_error &error)
{
    const QString message = error.message ? QString::fromUtf8(error.message)
                                          : QStringLiteral("unknown fakefs error");
    return QStringLiteral("%1 (line %2, code %3)").arg(message).arg(error.line).arg(error.code);
}

} // namespace

RootModel::RootModel(QObject *parent, bool managesRoots)
    : QAbstractListModel(parent)
    , m_managesRoots(managesRoots)
{
    const QString appData = QStandardPaths::writableLocation(QStandardPaths::AppDataLocation);
    if (!appData.isEmpty())
        m_rootPath = QDir(appData).filePath(QStringLiteral("roots"));
    QDir().mkpath(m_rootPath);
    if (m_managesRoots)
        m_defaultRoot = QSettings().value(QString::fromLatin1(kDefaultRootKey)).toString();
    refresh();
}

void RootModel::setRootPath(const QString &path)
{
    const QString clean = QDir::cleanPath(path);
    if (m_rootPath == clean)
        return;
    m_rootPath = clean;
    QDir().mkpath(m_rootPath);
    emit rootPathChanged();
    refresh();
}

void RootModel::setDefaultRoot(const QString &name)
{
    if (!m_managesRoots)
        return;
    if (!name.isEmpty() && !isRootNameValid(name)) {
        emit operationError(QStringLiteral("Invalid default root name: %1").arg(name));
        return;
    }
    if (!name.isEmpty() && !QDir(rootDirectoryForName(name)).exists()) {
        emit operationError(QStringLiteral("Root does not exist: %1").arg(name));
        return;
    }
    if (m_defaultRoot == name)
        return;
    m_defaultRoot = name;
    QSettings().setValue(QString::fromLatin1(kDefaultRootKey), m_defaultRoot);
    emit defaultRootChanged();
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
            if (!info.isDir())
                continue;
            m_entries.append({info.absoluteFilePath(), info.fileName(), true, info.size()});
        }
    }
    endResetModel();

    if (!m_managesRoots)
        return;
    if (!m_defaultRoot.isEmpty() && !QDir(rootDirectoryForName(m_defaultRoot)).exists()) {
        m_defaultRoot.clear();
        QSettings().remove(QString::fromLatin1(kDefaultRootKey));
    }
    if (m_defaultRoot.isEmpty() && !m_entries.isEmpty()) {
        m_defaultRoot = m_entries.first().name;
        QSettings().setValue(QString::fromLatin1(kDefaultRootKey), m_defaultRoot);
    }
    emit rootsChanged();
    emit defaultRootChanged();
}

bool RootModel::isRootNameValid(const QString &name) const
{
    return !name.isEmpty() && name != QStringLiteral(".") && name != QStringLiteral("..")
           && !name.contains(QLatin1Char('/')) && !name.contains(QLatin1Char('\\'))
           && !name.contains(QDir::separator());
}

QString RootModel::rootDirectoryForName(const QString &name) const
{
    return QDir(m_rootPath).filePath(name);
}

bool RootModel::ensureRootDirectory(QString *error) const
{
    if (m_rootPath.isEmpty()) {
        if (error)
            *error = QStringLiteral("Root storage directory is unavailable");
        return false;
    }
    if (QDir().mkpath(m_rootPath))
        return true;
    if (error)
        *error = QStringLiteral("Unable to create root storage directory: %1").arg(m_rootPath);
    return false;
}

void RootModel::progressCallback(void *cookie, double progress, const char *message, bool *shouldCancel)
{
    auto *self = static_cast<RootModel *>(cookie);
    if (shouldCancel)
        *shouldCancel = false;
    if (!self)
        return;
    const int percent = qBound(0, static_cast<int>(progress * 100.0), 100);
    emit self->operationProgress(percent, message ? QString::fromUtf8(message) : QString());
}

bool RootModel::reportFakefsError(const QString &operation, const struct fakefsify_error &error)
{
    const QString message = QStringLiteral("%1 failed: %2").arg(operation, fakefsErrorText(error));
    emit operationError(message);
    if (error.message)
        free(error.message);
    return false;
}

bool RootModel::importArchive(const QString &archivePath, const QString &name)
{
#if defined(Q_OS_ANDROID)
    Q_UNUSED(archivePath);
    Q_UNUSED(name);
    emit operationError(QStringLiteral("Root archive import is not available in this Android build"));
    return false;
#else
    QString error;
    if (!ensureRootDirectory(&error)) {
        emit operationError(error);
        return false;
    }
    if (!isRootNameValid(name)) {
        emit operationError(QStringLiteral("Filesystem name must not be empty or contain path separators"));
        return false;
    }
    if (QDir(rootDirectoryForName(name)).exists()) {
        emit operationError(QStringLiteral("Filesystem already exists: %1").arg(name));
        return false;
    }
    if (!QFileInfo::exists(archivePath)) {
        emit operationError(QStringLiteral("Archive does not exist: %1").arg(archivePath));
        return false;
    }

    const QString temporary = QDir(m_rootPath).filePath(QStringLiteral(".root-import-%1").arg(QDateTime::currentMSecsSinceEpoch()));
    QDir(temporary).removeRecursively();
    struct fakefsify_error fsError {};
    const bool ok = fakefs_import(QFileInfo(archivePath).absoluteFilePath().toUtf8().constData(),
                                  temporary.toUtf8().constData(), &fsError,
                                  {this, &RootModel::progressCallback});
    if (!ok) {
        QDir(temporary).removeRecursively();
        return reportFakefsError(QStringLiteral("Import"), fsError);
    }
    if (!QDir().rename(temporary, rootDirectoryForName(name))) {
        QDir(temporary).removeRecursively();
        emit operationError(QStringLiteral("Unable to install imported filesystem: %1").arg(name));
        return false;
    }
    refresh();
    emit operationProgress(100, QStringLiteral("Filesystem imported"));
    return true;
#endif
}

bool RootModel::exportArchive(const QString &name, const QString &archivePath)
{
#if defined(Q_OS_ANDROID)
    Q_UNUSED(name);
    Q_UNUSED(archivePath);
    emit operationError(QStringLiteral("Root archive export is not available in this Android build"));
    return false;
#else
    const QString source = rootDirectoryForName(name);
    if (!isRootNameValid(name) || !QDir(source).exists()) {
        emit operationError(QStringLiteral("Filesystem does not exist: %1").arg(name));
        return false;
    }
    if (archivePath.isEmpty()) {
        emit operationError(QStringLiteral("Archive path is empty"));
        return false;
    }

    struct fakefsify_error fsError {};
    const bool ok = fakefs_export(source.toUtf8().constData(),
                                  QFileInfo(archivePath).absoluteFilePath().toUtf8().constData(),
                                  &fsError, {this, &RootModel::progressCallback});
    if (!ok)
        return reportFakefsError(QStringLiteral("Export"), fsError);
    emit operationProgress(100, QStringLiteral("Filesystem exported"));
    return true;
#endif
}

bool RootModel::destroyRoot(const QString &name)
{
    if (!isRootNameValid(name) || !QDir(rootDirectoryForName(name)).exists()) {
        emit operationError(QStringLiteral("Filesystem does not exist: %1").arg(name));
        return false;
    }
    if (name == m_defaultRoot) {
        emit operationError(QStringLiteral("Cannot delete the default filesystem"));
        return false;
    }
    if (!QDir(rootDirectoryForName(name)).removeRecursively()) {
        emit operationError(QStringLiteral("Unable to delete filesystem: %1").arg(name));
        return false;
    }
    refresh();
    return true;
}

bool RootModel::renameRoot(const QString &name, const QString &newName)
{
    if (!isRootNameValid(name) || !QDir(rootDirectoryForName(name)).exists()) {
        emit operationError(QStringLiteral("Filesystem does not exist: %1").arg(name));
        return false;
    }
    if (!isRootNameValid(newName)) {
        emit operationError(QStringLiteral("New filesystem name is invalid"));
        return false;
    }
    if (name == m_defaultRoot) {
        emit operationError(QStringLiteral("Cannot rename the default filesystem"));
        return false;
    }
    if (QDir(rootDirectoryForName(newName)).exists()) {
        emit operationError(QStringLiteral("Filesystem already exists: %1").arg(newName));
        return false;
    }
    if (!QDir().rename(rootDirectoryForName(name), rootDirectoryForName(newName))) {
        emit operationError(QStringLiteral("Unable to rename filesystem"));
        return false;
    }
    refresh();
    return true;
}
