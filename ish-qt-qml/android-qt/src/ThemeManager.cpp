#include "ThemeManager.h"

#include <QColor>
#include <QDir>
#include <QFile>
#include <QFileSystemWatcher>
#include <QGuiApplication>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QPalette>
#include <QSaveFile>
#include <QStandardPaths>
#include <QStyleHints>

namespace {
constexpr int kThemeVersion = 1;

QVariantMap mapWithPalette(const QVariantMap &value)
{
    QVariantMap result = value;
    if (!result.contains(QStringLiteral("version")))
        result.insert(QStringLiteral("version"), kThemeVersion);
    return result;
}

QVariantMap selectedPalette(const QVariantMap &theme)
{
    const QVariantMap appearance = theme.value(QStringLiteral("appearance")).toMap();
    const bool lightOverride = appearance.value(QStringLiteral("lightOverride"), false).toBool();
    const bool darkOverride = appearance.value(QStringLiteral("darkOverride"), false).toBool();

    QVariantMap light = theme.value(QStringLiteral("light")).toMap();
    QVariantMap dark = theme.value(QStringLiteral("dark")).toMap();
    const QVariantMap shared = theme.value(QStringLiteral("shared")).toMap();
    if (light.isEmpty())
        light = shared;
    if (dark.isEmpty())
        dark = shared;

    bool systemDark = false;
#if QT_VERSION >= QT_VERSION_CHECK(6, 5, 0)
    if (QGuiApplication::styleHints())
        systemDark = QGuiApplication::styleHints()->colorScheme() == Qt::ColorScheme::Dark;
#else
    if (qApp)
        systemDark = qApp->palette().color(QPalette::Window).lightness() < 128;
#endif
    if (lightOverride)
        return dark;
    if (darkOverride)
        return light;
    return systemDark ? dark : light;
}

QString paletteSelectionColor(const QVariantMap &palette)
{
    const QColor background(palette.value(QStringLiteral("backgroundColor")).toString());
    const QColor foreground(palette.value(QStringLiteral("foregroundColor")).toString());
    if (!background.isValid() || !foreground.isValid())
        return QStringLiteral("#264f78");
    QColor selection = background;
    selection.setRedF((background.redF() * 0.55) + (foreground.redF() * 0.45));
    selection.setGreenF((background.greenF() * 0.55) + (foreground.greenF() * 0.45));
    selection.setBlueF((background.blueF() * 0.55) + (foreground.blueF() * 0.45));
    return selection.name(QColor::HexRgb);
}
}

ThemeManager::ThemeManager(QObject *parent)
    : QObject(parent),
      m_themesDirectory(QDir(QStandardPaths::writableLocation(QStandardPaths::AppDataLocation))
                            .filePath(QStringLiteral("themes"))),
      m_watcher(new QFileSystemWatcher(this))
{
    QDir().mkpath(m_themesDirectory);
    installDefaultThemes();
    connect(m_watcher, &QFileSystemWatcher::directoryChanged,
            this, &ThemeManager::reloadUserThemes);
    m_watcher->addPath(m_themesDirectory);
    reloadUserThemes();
}

QVariantMap ThemeManager::palette(const QString &foreground,
                                  const QString &background,
                                  const QString &cursor,
                                  const QStringList &overrides)
{
    QVariantMap result{
        {QStringLiteral("foregroundColor"), foreground},
        {QStringLiteral("backgroundColor"), background},
    };
    if (!cursor.isEmpty())
        result.insert(QStringLiteral("cursorColor"), cursor);
    if (!overrides.isEmpty())
        result.insert(QStringLiteral("colorPaletteOverrides"), overrides);
    return result;
}

QVariantMap ThemeManager::appearance(bool lightOverride, bool darkOverride)
{
    return {
        {QStringLiteral("lightOverride"), lightOverride},
        {QStringLiteral("darkOverride"), darkOverride},
    };
}

void ThemeManager::installDefaultThemes()
{
    const QStringList solarized{
        QStringLiteral("#073642"), QStringLiteral("#dc322f"), QStringLiteral("#859900"),
        QStringLiteral("#b58900"), QStringLiteral("#268bd2"), QStringLiteral("#d33682"),
        QStringLiteral("#2aa198"), QStringLiteral("#eee8d5"), QStringLiteral("#002b36"),
        QStringLiteral("#cb4b16"), QStringLiteral("#586e75"), QStringLiteral("#657b83"),
        QStringLiteral("#839496"), QStringLiteral("#6c71c4"), QStringLiteral("#93a1a1"),
        QStringLiteral("#fdf6e3")
    };

    const QVariantMap defaultTheme{
        {QStringLiteral("version"), kThemeVersion},
        {QStringLiteral("light"), palette(QStringLiteral("#000"), QStringLiteral("#fff"))},
        {QStringLiteral("dark"), palette(QStringLiteral("#fff"), QStringLiteral("#000"))},
    };
    const QVariantMap hackerTheme{
        {QStringLiteral("version"), kThemeVersion},
        {QStringLiteral("shared"), palette(QStringLiteral("#0f0"), QStringLiteral("#000"))},
        {QStringLiteral("appearance"), appearance(true, false)},
    };
    const QVariantMap solarizedTheme{
        {QStringLiteral("version"), kThemeVersion},
        {QStringLiteral("light"), palette(QStringLiteral("#657b83"), QStringLiteral("#fdf6e3"), {}, solarized)},
        {QStringLiteral("dark"), palette(QStringLiteral("#839496"), QStringLiteral("#002b36"), {}, solarized)},
    };
    const QVariantMap hotDogTheme{
        {QStringLiteral("version"), kThemeVersion},
        {QStringLiteral("shared"), palette(QStringLiteral("#ff0"), QStringLiteral("#f00"))},
    };

    m_defaultThemes = {
        {QStringLiteral("Default"), defaultTheme, false},
        {QStringLiteral("1337"), hackerTheme, false},
        {QStringLiteral("Solarized"), solarizedTheme, false},
        {QStringLiteral("Hot Dog Stand"), hotDogTheme, false},
    };
}

bool ThemeManager::validColor(const QVariant &value)
{
    const QString color = value.toString();
    if (color.isEmpty() || !color.startsWith(QLatin1Char('#')) ||
        (color.size() != 4 && color.size() != 5 && color.size() != 7 && color.size() != 9))
        return false;
    return QColor(color).isValid();
}

bool ThemeManager::validPalette(const QVariantMap &value)
{
    if (!validColor(value.value(QStringLiteral("foregroundColor"))) ||
        !validColor(value.value(QStringLiteral("backgroundColor"))))
        return false;
    const QVariant cursor = value.value(QStringLiteral("cursorColor"));
    if (!cursor.isNull() && !cursor.toString().isEmpty() && !validColor(cursor))
        return false;
    const QVariant overrides = value.value(QStringLiteral("colorPaletteOverrides"));
    if (!overrides.isValid())
        return true;
    const QStringList colors = overrides.toStringList();
    if (colors.size() != 16)
        return false;
    for (const QString &color : colors) {
        if (!validColor(color))
            return false;
    }
    return true;
}

QVariantMap ThemeManager::normalizeTheme(const QVariantMap &value)
{
    QVariantMap result = mapWithPalette(value);
    const int version = result.value(QStringLiteral("version")).toInt();
    if (version <= 0 || version > kThemeVersion)
        return {};

    if (result.contains(QStringLiteral("shared"))) {
        if (!validPalette(result.value(QStringLiteral("shared")).toMap()))
            return {};
        result.remove(QStringLiteral("light"));
        result.remove(QStringLiteral("dark"));
    } else if (result.contains(QStringLiteral("light")) && result.contains(QStringLiteral("dark"))) {
        if (!validPalette(result.value(QStringLiteral("light")).toMap()) ||
            !validPalette(result.value(QStringLiteral("dark")).toMap()))
            return {};
        result.remove(QStringLiteral("shared"));
    } else if (validPalette(result)) {
        const QVariantMap shared = result;
        result.clear();
        result.insert(QStringLiteral("version"), kThemeVersion);
        result.insert(QStringLiteral("shared"), shared);
    } else {
        return {};
    }

    if (result.contains(QStringLiteral("appearance"))) {
        const QVariantMap a = result.value(QStringLiteral("appearance")).toMap();
        if (!a.contains(QStringLiteral("lightOverride")) ||
            !a.contains(QStringLiteral("darkOverride")))
            return {};
    }
    return result;
}

QByteArray ThemeManager::serializeTheme(const QVariantMap &theme)
{
    const QJsonObject object = QJsonObject::fromVariantMap(theme);
    return QJsonDocument(object).toJson(QJsonDocument::Indented);
}

QVariantMap ThemeManager::parseTheme(const QByteArray &data)
{
    QJsonParseError error{};
    const QJsonDocument document = QJsonDocument::fromJson(data, &error);
    if (error.error != QJsonParseError::NoError || !document.isObject())
        return {};
    return normalizeTheme(document.object().toVariantMap());
}

QString ThemeManager::userThemePath(const QString &name) const
{
    return QDir(m_themesDirectory).filePath(name + QStringLiteral(".json"));
}

QString ThemeManager::uniqueDuplicateName(const QString &name) const
{
    for (int suffix = 1;; ++suffix) {
        const QString candidate = QStringLiteral("%1-%2").arg(name).arg(suffix);
        if (!m_themeNames.contains(candidate))
            return candidate;
    }
}

void ThemeManager::updateThemeNames()
{
    QStringList names;
    for (const ThemeRecord &theme : std::as_const(m_userThemes))
        names.append(theme.name);
    for (const ThemeRecord &theme : std::as_const(m_defaultThemes))
        if (!names.contains(theme.name))
            names.append(theme.name);
    if (names == m_themeNames)
        return;
    m_themeNames = names;
    emit themeNamesChanged();
}

void ThemeManager::reloadUserThemes()
{
    QList<ThemeRecord> themes;
    const QFileInfoList files = QDir(m_themesDirectory).entryInfoList(
        {QStringLiteral("*.json")}, QDir::Files | QDir::Readable, QDir::Name);
    for (const QFileInfo &file : files) {
        QFile input(file.absoluteFilePath());
        if (!input.open(QIODevice::ReadOnly))
            continue;
        const QVariantMap representation = parseTheme(input.readAll());
        if (!representation.isEmpty())
            themes.append({file.completeBaseName(), representation, true});
    }
    m_userThemes = themes;
    updateThemeNames();
    emit themesUpdated();
}

QVariantMap ThemeManager::themeForName(const QString &name) const
{
    for (const ThemeRecord &theme : m_userThemes)
        if (theme.name == name)
            return theme.representation;
    for (const ThemeRecord &theme : m_defaultThemes)
        if (theme.name == name)
            return theme.representation;
    return {};
}

QVariantMap ThemeManager::styleForName(const QString &name) const
{
    QVariantMap selected = selectedPalette(themeForName(name));
    if (selected.isEmpty())
        selected = selectedPalette(themeForName(QStringLiteral("Default")));

    QVariantMap style{
        {QStringLiteral("backgroundColor"), selected.value(QStringLiteral("backgroundColor"))},
        {QStringLiteral("foregroundColor"), selected.value(QStringLiteral("foregroundColor"))},
        {QStringLiteral("cursorColor"), selected.value(QStringLiteral("cursorColor"),
                                                         selected.value(QStringLiteral("foregroundColor")))},
        {QStringLiteral("selectionColor"), paletteSelectionColor(selected)},
        {QStringLiteral("colorPaletteOverrides"), selected.value(QStringLiteral("colorPaletteOverrides"))},
    };
    return style;
}

bool ThemeManager::writeUserTheme(const QString &name, const QVariantMap &theme)
{
    const QVariantMap normalized = normalizeTheme(theme);
    if (normalized.isEmpty()) {
        emit themeError(QStringLiteral("Invalid theme palette or version"));
        return false;
    }
    QSaveFile output(userThemePath(name));
    if (!output.open(QIODevice::WriteOnly) || output.write(serializeTheme(normalized)) < 0 ||
        !output.commit()) {
        emit themeError(QStringLiteral("Unable to save theme %1").arg(name));
        return false;
    }
    return true;
}

bool ThemeManager::addUserTheme(const QString &name, const QVariantMap &theme)
{
    const QString cleanName = name.trimmed();
    if (cleanName.isEmpty() || cleanName.contains(QLatin1Char('/')) || m_themeNames.contains(cleanName))
        return false;
    if (!writeUserTheme(cleanName, theme))
        return false;
    reloadUserThemes();
    return true;
}

bool ThemeManager::replaceUserTheme(const QString &oldName,
                                    const QString &newName,
                                    const QVariantMap &theme)
{
    const QString oldClean = oldName.trimmed();
    const QString newClean = newName.trimmed();
    if (oldClean.isEmpty() || newClean.isEmpty() || newClean.contains(QLatin1Char('/')) ||
        !isUserTheme(oldClean) || (oldClean != newClean && m_themeNames.contains(newClean)))
        return false;
    if (!writeUserTheme(newClean, theme))
        return false;
    if (oldClean != newClean)
        QFile::remove(userThemePath(oldClean));
    reloadUserThemes();
    return true;
}

bool ThemeManager::duplicateUserTheme(const QString &name)
{
    const QVariantMap source = themeForName(name);
    if (source.isEmpty())
        return false;
    return addUserTheme(uniqueDuplicateName(name), source);
}

bool ThemeManager::deleteUserTheme(const QString &name)
{
    if (!isUserTheme(name) || !QFile::remove(userThemePath(name)))
        return false;
    reloadUserThemes();
    return true;
}

bool ThemeManager::isUserTheme(const QString &name) const
{
    for (const ThemeRecord &theme : m_userThemes)
        if (theme.name == name)
            return true;
    return false;
}
