#pragma once

#include <QObject>
#include <QStringList>
#include <QVariantMap>

class QFileSystemWatcher;

class ThemeManager final : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QStringList themeNames READ themeNames NOTIFY themeNamesChanged)
    Q_PROPERTY(QString themesDirectory READ themesDirectory CONSTANT)

public:
    explicit ThemeManager(QObject *parent = nullptr);

    QStringList themeNames() const { return m_themeNames; }
    QString themesDirectory() const { return m_themesDirectory; }

    Q_INVOKABLE QVariantMap styleForName(const QString &name) const;
    Q_INVOKABLE QVariantMap themeForName(const QString &name) const;
    Q_INVOKABLE bool addUserTheme(const QString &name, const QVariantMap &theme);
    Q_INVOKABLE bool replaceUserTheme(const QString &oldName,
                                      const QString &newName,
                                      const QVariantMap &theme);
    Q_INVOKABLE bool duplicateUserTheme(const QString &name);
    Q_INVOKABLE bool deleteUserTheme(const QString &name);
    Q_INVOKABLE bool isUserTheme(const QString &name) const;

signals:
    void themeNamesChanged();
    void themesUpdated();
    void themeError(const QString &message);

private slots:
    void reloadUserThemes();

private:
    struct ThemeRecord {
        QString name;
        QVariantMap representation;
        bool userTheme = false;
    };

    static QVariantMap palette(const QString &foreground,
                               const QString &background,
                               const QString &cursor = {},
                               const QStringList &overrides = {});
    static QVariantMap appearance(bool lightOverride, bool darkOverride);
    static bool validColor(const QVariant &value);
    static bool validPalette(const QVariantMap &value);
    static QVariantMap normalizeTheme(const QVariantMap &value);
    static QByteArray serializeTheme(const QVariantMap &theme);
    static QVariantMap parseTheme(const QByteArray &data);

    QString userThemePath(const QString &name) const;
    QString uniqueDuplicateName(const QString &name) const;
    void installDefaultThemes();
    void updateThemeNames();
    bool writeUserTheme(const QString &name, const QVariantMap &theme);

    QString m_themesDirectory;
    QList<ThemeRecord> m_defaultThemes;
    QList<ThemeRecord> m_userThemes;
    QStringList m_themeNames;
    QFileSystemWatcher *m_watcher = nullptr;
};
