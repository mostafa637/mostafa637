#include "RootfsManager.h"

#include <QDir>
#include <QFile>
#include <QFileInfo>
#include <QSaveFile>
#include <QStandardPaths>
#include <QUrl>
#include <QUrlQuery>
#include <QDebug>

#include <algorithm>
#include <cerrno>
#include <cstring>
#include <memory>

#include <sqlite3.h>
#include <zlib.h>

namespace {

struct IshStat {
    quint32 mode = 0;
    quint32 uid = 0;
    quint32 gid = 0;
    quint32 rdev = 0;
};

struct TarEntry {
    QByteArray path;
    QByteArray link;
    quint64 size = 0;
    quint32 mode = 0;
    quint32 uid = 0;
    quint32 gid = 0;
    char type = '0';
};

QByteArray field(const char *data, int length)
{
    int end = 0;
    while (end < length && data[end] != '\0')
        ++end;
    return QByteArray(data, end).trimmed();
}

quint64 octalField(const char *data, int length)
{
    QByteArray value = field(data, length);
    value = value.trimmed();
    if (value.isEmpty())
        return 0;
    bool ok = false;
    const quint64 result = value.toULongLong(&ok, 8);
    return ok ? result : 0;
}

bool isZeroBlock(const QByteArray &block)
{
    for (char c : block) {
        if (c != '\0')
            return false;
    }
    return true;
}

bool normalizeTarPath(const QByteArray &raw, QByteArray *out)
{
    QByteArray path = raw;
    while (path.startsWith("./"))
        path.remove(0, 2);
    while (path.startsWith('/'))
        path.remove(0, 1);
    while (path.endsWith('/'))
        path.chop(1);

    QList<QByteArray> components;
    for (const QByteArray &component : path.split('/')) {
        if (component.isEmpty() || component == ".")
            continue;
        if (component == "..")
            return false;
        components.append(component);
    }
    *out = components.join('/');
    return true;
}

QString displayPath(const QByteArray &path)
{
    return path.isEmpty() ? QStringLiteral("/") : QStringLiteral("/") + QString::fromUtf8(path);
}

bool readExact(gzFile file, char *buffer, int length)
{
    int read = 0;
    while (read < length) {
        const int count = gzread(file, buffer + read, length - read);
        if (count <= 0)
            return false;
        read += count;
    }
    return true;
}

bool skipExact(gzFile file, quint64 length)
{
    QByteArray buffer(8192, Qt::Uninitialized);
    while (length > 0) {
        const unsigned chunk = static_cast<unsigned>(std::min<quint64>(length, buffer.size()));
        if (!readExact(file, buffer.data(), static_cast<int>(chunk)))
            return false;
        length -= chunk;
    }
    return true;
}

QFile::Permissions permissionsFromMode(quint32 mode)
{
    QFile::Permissions permissions;
    if (mode & 0400) permissions |= QFileDevice::ReadOwner;
    if (mode & 0200) permissions |= QFileDevice::WriteOwner;
    if (mode & 0100) permissions |= QFileDevice::ExeOwner;
    if (mode & 0040) permissions |= QFileDevice::ReadGroup;
    if (mode & 0020) permissions |= QFileDevice::WriteGroup;
    if (mode & 0010) permissions |= QFileDevice::ExeGroup;
    if (mode & 0004) permissions |= QFileDevice::ReadOther;
    if (mode & 0002) permissions |= QFileDevice::WriteOther;
    if (mode & 0001) permissions |= QFileDevice::ExeOther;
    return permissions;
}

QString sqliteError(sqlite3 *db, const QString &fallback)
{
    if (db && sqlite3_errmsg(db))
        return QStringLiteral("%1: %2").arg(fallback, QString::fromUtf8(sqlite3_errmsg(db)));
    return fallback;
}

} // namespace

RootfsManager::RootfsManager(QObject *parent)
    : QObject(parent)
{
    m_rootPath = QDir(QStandardPaths::writableLocation(QStandardPaths::AppDataLocation))
                     .filePath(QStringLiteral("rootfs"));
    refreshRepositoryState();
}

void RootfsManager::prepare()
{
    if (m_prepared)
        return;

    const QString appData = QStandardPaths::writableLocation(QStandardPaths::AppDataLocation);
    if (appData.isEmpty()) {
        emit preparationError(QStringLiteral("Application data directory is unavailable"));
        return;
    }
    QDir().mkpath(appData);
    QDir().mkpath(m_rootPath);

    const QString archivePath = QDir(appData).filePath(QStringLiteral("root.tar.gz"));
    QFile archive(archivePath);
    if (!archive.exists()) {
        QFile resource(QStringLiteral(":/ish-assets/rootfs/root.tar.gz"));
        if (!resource.open(QIODevice::ReadOnly) || !archive.open(QIODevice::WriteOnly)) {
            emit preparationError(QStringLiteral("Unable to copy bundled rootfs"));
            return;
        }
        while (!resource.atEnd()) {
            const QByteArray chunk = resource.read(1024 * 1024);
            if (chunk.isEmpty() && !resource.atEnd()) {
                emit preparationError(QStringLiteral("Unable to read bundled rootfs"));
                return;
            }
            if (archive.write(chunk) != chunk.size()) {
                emit preparationError(QStringLiteral("Unable to write bundled rootfs"));
                return;
            }
        }
        archive.close();
        resource.close();
    }

    emit progressChanged(1, QStringLiteral("Preparing bundled rootfs"));
    QString error;
    if (!importBundledRootfs(archivePath, m_rootPath, &error)) {
        emit preparationError(error);
        return;
    }
    QFile::remove(archivePath);
    refreshRepositoryState();
    setPrepared(true);
    emit progressChanged(100, QStringLiteral("Rootfs is ready"));
}

void RootfsManager::resetInstalledData()
{
    if (m_prepared)
        setPrepared(false);
    QDir(m_rootPath).removeRecursively();
    refreshRepositoryState();
}

QString RootfsManager::terminalUrl(const QString &webChannelUrl) const
{
    QUrl url(QStringLiteral("qrc:/ish-assets/terminal/term.html"));
    if (!webChannelUrl.isEmpty()) {
        QUrlQuery query;
        query.addQueryItem(QStringLiteral("ws"), webChannelUrl);
        url.setQuery(query);
    }
    return url.toString();
}

bool RootfsManager::importBundledRootfs(const QString &archivePath,
                                         const QString &destination,
                                         QString *error)
{
    QDir destinationDir(destination);
    if (QFileInfo::exists(destination))
        destinationDir.removeRecursively();
    if (!QDir().mkpath(QDir(destination).filePath(QStringLiteral("data")))) {
        if (error) *error = QStringLiteral("Unable to create fakefs data directory");
        return false;
    }

    gzFile archive = gzopen(QFile::encodeName(archivePath).constData(), "rb");
    if (!archive) {
        if (error) *error = QStringLiteral("Unable to open bundled rootfs archive");
        return false;
    }

    const qint64 archiveSize = QFileInfo(archivePath).size();
    qint64 approximateRead = 0;
    QList<QByteArray> metadataPaths;
    QList<quint32> metadataModes;
    QList<quint32> metadataUids;
    QList<quint32> metadataGids;
    bool rootSeen = false;
    bool ok = true;
    QString localError;

    QByteArray header(512, Qt::Uninitialized);
    while (ok) {
        if (!readExact(archive, header.data(), header.size())) {
            localError = QStringLiteral("Truncated rootfs tar header");
            ok = false;
            break;
        }
        if (isZeroBlock(header))
            break;

        TarEntry entry;
        QByteArray name = field(header.constData(), 100);
        const QByteArray prefix = field(header.constData() + 345, 155);
        if (!prefix.isEmpty())
            name = prefix + '/' + name;
        if (!normalizeTarPath(name, &entry.path) ||
            !normalizeTarPath(field(header.constData() + 157, 100), &entry.link)) {
            localError = QStringLiteral("Rootfs contains an unsafe path");
            ok = false;
            break;
        }
        entry.size = octalField(header.constData() + 124, 12);
        entry.mode = static_cast<quint32>(octalField(header.constData() + 100, 8));
        entry.uid = static_cast<quint32>(octalField(header.constData() + 108, 8));
        entry.gid = static_cast<quint32>(octalField(header.constData() + 116, 8));
        entry.type = header[156] == '\0' ? '0' : header[156];
        if (entry.type == '5')
            entry.mode |= 0040000;
        else if (entry.type == '2')
            entry.mode |= 0120000;
        else
            entry.mode |= 0100000;

        const QString relative = QString::fromUtf8(entry.path);
        const QString outputPath = QDir(destination).filePath(QStringLiteral("data/") + relative);
        if (entry.path.isEmpty()) {
            rootSeen = true;
        } else if (entry.type == '5') {
            if (!QDir().mkpath(outputPath)) {
                localError = QStringLiteral("Unable to create rootfs directory %1").arg(displayPath(entry.path));
                ok = false;
            }
        } else if (entry.type == '2') {
            QDir().mkpath(QFileInfo(outputPath).absolutePath());
            QFile linkFile(outputPath);
            if (!linkFile.open(QIODevice::WriteOnly) || linkFile.write(entry.link) != entry.link.size()) {
                localError = QStringLiteral("Unable to create rootfs link %1").arg(displayPath(entry.path));
                ok = false;
            }
            linkFile.close();
        } else {
            QDir().mkpath(QFileInfo(outputPath).absolutePath());
            QFile output(outputPath);
            if (!output.open(QIODevice::WriteOnly | QIODevice::Truncate)) {
                localError = QStringLiteral("Unable to create rootfs file %1").arg(displayPath(entry.path));
                ok = false;
            } else {
                quint64 remaining = entry.size;
                QByteArray buffer(1024 * 1024, Qt::Uninitialized);
                while (remaining > 0 && ok) {
                    const int chunk = static_cast<int>(std::min<quint64>(remaining, buffer.size()));
                    if (!readExact(archive, buffer.data(), chunk) || output.write(buffer.constData(), chunk) != chunk) {
                        localError = QStringLiteral("Unable to extract rootfs file %1").arg(displayPath(entry.path));
                        ok = false;
                        break;
                    }
                    remaining -= static_cast<quint64>(chunk);
                }
                output.close();
                if (ok)
                    QFile::setPermissions(outputPath, permissionsFromMode(entry.mode));
            }
        }

        if (entry.type != '0' && entry.type != '\0' && entry.size > 0 &&
            !skipExact(archive, entry.size)) {
            localError = QStringLiteral("Unable to skip rootfs entry payload");
            ok = false;
        }
        const quint64 padding = (512 - (entry.size % 512)) % 512;
        if (ok && padding > 0 && !skipExact(archive, padding)) {
            localError = QStringLiteral("Unable to read rootfs tar padding");
            ok = false;
        }

        if (ok) {
            metadataPaths.append(entry.path);
            metadataModes.append(entry.mode);
            metadataUids.append(entry.uid);
            metadataGids.append(entry.gid);
            approximateRead += 512 + static_cast<qint64>(entry.size);
            const int percent = archiveSize > 0
                ? std::clamp(static_cast<int>((approximateRead * 95) / archiveSize), 2, 95)
                : 50;
            emit progressChanged(percent, QStringLiteral("Importing %1").arg(displayPath(entry.path)));
        }
    }
    gzclose(archive);

    if (ok && !rootSeen) {
        metadataPaths.append(QByteArray());
        metadataModes.append(0040755);
        metadataUids.append(0);
        metadataGids.append(0);
    }
    if (ok && !writeMetadata(destination, metadataPaths, metadataModes, metadataUids, metadataGids, &localError))
        ok = false;

    if (!ok) {
        QDir(destination).removeRecursively();
        if (error) *error = localError.isEmpty() ? QStringLiteral("Rootfs import failed") : localError;
    }
    return ok;
}

bool RootfsManager::writeMetadata(const QString &destination,
                                   const QList<QByteArray> &paths,
                                   const QList<quint32> &modes,
                                   const QList<quint32> &uids,
                                   const QList<quint32> &gids,
                                   QString *error)
{
    const QString dbPath = QDir(destination).filePath(QStringLiteral("meta.db"));
    sqlite3 *db = nullptr;
    if (sqlite3_open_v2(QFile::encodeName(dbPath).constData(), &db,
                        SQLITE_OPEN_READWRITE | SQLITE_OPEN_CREATE, nullptr) != SQLITE_OK) {
        if (error) *error = sqliteError(db, QStringLiteral("Unable to create fakefs metadata"));
        if (db) sqlite3_close(db);
        return false;
    }

    const char *schema =
        "PRAGMA journal_mode=DELETE;"
        "CREATE TABLE meta (id integer unique default 0, db_inode integer);"
        "INSERT INTO meta (db_inode) VALUES (0);"
        "CREATE TABLE stats (inode integer primary key, stat blob);"
        "CREATE TABLE paths (path blob primary key, inode integer references stats(inode));"
        "CREATE INDEX inode_to_path ON paths (inode, path);"
        "PRAGMA user_version=3;";
    char *sqliteMessage = nullptr;
    bool ok = sqlite3_exec(db, "BEGIN;", nullptr, nullptr, &sqliteMessage) == SQLITE_OK;
    if (ok)
        ok = sqlite3_exec(db, schema, nullptr, nullptr, &sqliteMessage) == SQLITE_OK;

    sqlite3_stmt *insertStat = nullptr;
    sqlite3_stmt *insertPath = nullptr;
    if (ok)
        ok = sqlite3_prepare_v2(db, "INSERT INTO stats (stat) VALUES (?)", -1, &insertStat, nullptr) == SQLITE_OK;
    if (ok)
        ok = sqlite3_prepare_v2(db, "INSERT OR REPLACE INTO paths (path, inode) VALUES (?, ?)", -1, &insertPath, nullptr) == SQLITE_OK;

    if (ok) {
        for (int i = 0; i < paths.size(); ++i) {
            IshStat stat{modes.value(i), uids.value(i), gids.value(i), 0};
            if (sqlite3_bind_blob(insertStat, 1, &stat, sizeof(stat), SQLITE_TRANSIENT) != SQLITE_OK ||
                sqlite3_step(insertStat) != SQLITE_DONE) {
                ok = false;
                break;
            }
            const sqlite3_int64 inode = sqlite3_last_insert_rowid(db);
            sqlite3_reset(insertStat);
            sqlite3_clear_bindings(insertStat);
            if (sqlite3_bind_blob(insertPath, 1, paths.at(i).constData(), paths.at(i).size(), SQLITE_TRANSIENT) != SQLITE_OK ||
                sqlite3_bind_int64(insertPath, 2, inode) != SQLITE_OK ||
                sqlite3_step(insertPath) != SQLITE_DONE) {
                ok = false;
                break;
            }
            sqlite3_reset(insertPath);
            sqlite3_clear_bindings(insertPath);
        }
    }

    if (insertStat) sqlite3_finalize(insertStat);
    if (insertPath) sqlite3_finalize(insertPath);
    if (ok)
        ok = sqlite3_exec(db, "COMMIT;", nullptr, nullptr, &sqliteMessage) == SQLITE_OK;
    else
        sqlite3_exec(db, "ROLLBACK;", nullptr, nullptr, nullptr);

    const QString dbError = sqliteError(db, QStringLiteral("Unable to write fakefs metadata"));
    if (sqliteMessage)
        sqlite3_free(sqliteMessage);
    sqlite3_close(db);
    if (!ok && error)
        *error = dbError;
    return ok;
}

void RootfsManager::refreshRepositoryState()
{
    const QString data = QDir(m_rootPath).filePath(QStringLiteral("data"));
    const bool managed = QFileInfo::exists(QDir(data).filePath(QStringLiteral("etc/apk/repositories")));
    if (m_repositoryManaged != managed) {
        m_repositoryManaged = managed;
        emit repositoryManagedChanged();
    }
    const bool updateRequired = false;
    if (m_repositoryUpdateRequired != updateRequired) {
        m_repositoryUpdateRequired = updateRequired;
        emit repositoryUpdateRequiredChanged();
    }
}

void RootfsManager::setPrepared(bool value)
{
    if (m_prepared == value)
        return;
    m_prepared = value;
    emit preparedChanged();
    if (value)
        refreshRepositoryState();
}
