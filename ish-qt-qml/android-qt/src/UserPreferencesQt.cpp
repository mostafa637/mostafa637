#include "UserPreferencesQt.h"
#include "ThemeManager.h"

#include <QCoreApplication>
#include <QDir>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonParseError>
#include <QJsonValue>
#include <QMetaObject>
#include <QMutex>
#include <QMutexLocker>
#include <QSettings>
#include <QStandardPaths>
#include <QSysInfo>

#include <algorithm>
#include <cstdlib>
#include <cstring>

extern "C" {
#include "fs/proc/ish.h"
}

namespace {

QSettings settings()
{
    return QSettings(QStringLiteral("iSH"), QStringLiteral("iSH Qt"));
}

struct PreferenceMapping {
    const char *friendly;
    const char *underlying;
};

constexpr PreferenceMapping kMappings[] = {
    {"caps_lock_mapping", "Caps Lock Mapping"},
    {"option_mapping", "Option Mapping"},
    {"backtick_mapping_escape", "Backtick Mapping Escape"},
    {"hide_extra_keys_with_external_keyboard", "Hide Extra Keys With External Keyboard"},
    {"override_control_space", "Override Control Space"},
    {"font_family", "Font Family"},
    {"font_size", "Font Size"},
    {"disable_dimming", "Disable Dimming"},
    {"launch_command", "Init Command"},
    {"boot_command", "Boot Command"},
    {"cursor_style", "Cursor Style"},
    {"blink_cursor", "Blink Cursor"},
    {"hide_status_bar", "Status Bar"},
    {"color_scheme", "Color Scheme"},
    {"theme", "ModernTheme"},
    {"hostname_override", "hostnameOverride"},
};

QMutex g_preferencesMutex;
UserPreferencesQt *g_preferences = nullptr;

const PreferenceMapping *mappingForFriendly(const QString &name)
{
    for (const auto &mapping : kMappings)
        if (name == QLatin1String(mapping.friendly))
            return &mapping;
    return nullptr;
}

const PreferenceMapping *mappingForUnderlying(const QString &name)
{
    for (const auto &mapping : kMappings)
        if (name == QLatin1String(mapping.underlying))
            return &mapping;
    return nullptr;
}

QVariant decodeJsonFragment(const char *buffer, size_t size, bool *ok)
{
    QByteArray wrapped("[");
    wrapped.append(buffer, static_cast<qsizetype>(size));
    wrapped.append(']');
    QJsonParseError error{};
    const QJsonDocument document = QJsonDocument::fromJson(wrapped, &error);
    if (error.error != QJsonParseError::NoError || !document.isArray() || document.array().isEmpty()) {
        if (ok)
            *ok = false;
        return {};
    }
    if (ok)
        *ok = true;
    return document.array().first().toVariant();
}

QByteArray encodeJsonFragment(const QVariant &value)
{
    QJsonArray array;
    array.append(QJsonValue::fromVariant(value));
    const QByteArray document = QJsonDocument(array).toJson(QJsonDocument::Compact);
    if (document.size() < 2)
        return {};
    return document.mid(1, document.size() - 2) + '\n';
}

bool validExternalValue(const QString &key, const QVariant &value)
{
    if (key == QStringLiteral("Caps Lock Mapping"))
        return value.canConvert<int>() && value.toInt() >= 0 && value.toInt() < 3;
    if (key == QStringLiteral("Option Mapping"))
        return value.canConvert<int>() && value.toInt() >= 0 && value.toInt() < 2;
    if (key == QStringLiteral("Cursor Style"))
        return value.canConvert<int>() && value.toInt() >= 0 && value.toInt() < 3;
    if (key == QStringLiteral("Color Scheme"))
        return value.canConvert<int>() && value.toInt() >= 0 && value.toInt() < 3;
    if (key == QStringLiteral("Font Size"))
        return value.canConvert<int>() && value.toInt() >= 6 && value.toInt() <= 72;
    if (key == QStringLiteral("Font Family") || key == QStringLiteral("ModernTheme") ||
        key == QStringLiteral("hostnameOverride"))
        return value.userType() == QMetaType::QString;
    if (key == QStringLiteral("Init Command") || key == QStringLiteral("Boot Command"))
        return value.userType() == QMetaType::QStringList || value.userType() == QMetaType::QVariantList;
    return value.userType() == QMetaType::Bool || value.canConvert<bool>();
}

void queuePreferenceReload()
{
    QMutexLocker locker(&g_preferencesMutex);
    if (g_preferences)
        QMetaObject::invokeMethod(g_preferences, "reload", Qt::QueuedConnection);
}

char **getAllDefaultsKeysImpl()
{
    const QStringList keys = settings().allKeys();
    char **entries = static_cast<char **>(std::calloc(static_cast<size_t>(keys.size()) + 1,
                                                       sizeof(*entries)));
    if (!entries)
        return nullptr;
    for (qsizetype i = 0; i < keys.size(); ++i) {
        const QByteArray utf8 = keys.at(i).toUtf8();
        entries[i] = ::strdup(utf8.constData());
        if (!entries[i]) {
            for (qsizetype j = 0; j < i; ++j)
                std::free(entries[j]);
            std::free(entries);
            return nullptr;
        }
    }
    return entries;
}

char *getFriendlyNameImpl(const char *name)
{
    if (!name)
        return nullptr;
    const PreferenceMapping *mapping = mappingForUnderlying(QString::fromUtf8(name));
    if (!mapping)
        return nullptr;
    return ::strdup(mapping->friendly);
}

char *getUnderlyingNameImpl(const char *name)
{
    if (!name)
        return nullptr;
    const PreferenceMapping *mapping = mappingForFriendly(QString::fromUtf8(name));
    return ::strdup(mapping ? mapping->underlying : name);
}

bool getUserDefaultImpl(const char *name, char **buffer, size_t *size)
{
    if (!name || !buffer || !size)
        return false;
    const QVariant value = settings().value(QString::fromUtf8(name));
    if (!value.isValid())
        return false;
    const QByteArray encoded = encodeJsonFragment(value);
    if (encoded.isEmpty())
        return false;
    char *copy = static_cast<char *>(std::malloc(static_cast<size_t>(encoded.size())));
    if (!copy)
        return false;
    std::memcpy(copy, encoded.constData(), static_cast<size_t>(encoded.size()));
    *buffer = copy;
    *size = static_cast<size_t>(encoded.size());
    return true;
}

bool setUserDefaultImpl(const char *name, char *buffer, size_t size)
{
    if (!name || !buffer || size == 0)
        return false;
    bool ok = false;
    const QString key = QString::fromUtf8(name);
    const QVariant value = decodeJsonFragment(buffer, size, &ok);
    if (!ok || !validExternalValue(key, value))
        return false;
    QSettings store = settings();
    store.setValue(key, value);
    store.sync();
    queuePreferenceReload();
    return store.status() == QSettings::NoError;
}

bool removeUserDefaultImpl(const char *name)
{
    if (!name)
        return false;
    QSettings store = settings();
    store.remove(QString::fromUtf8(name));
    store.sync();
    queuePreferenceReload();
    return store.status() == QSettings::NoError;
}

char *getDocumentsDirectoryImpl()
{
    const QString path = QDir(QStandardPaths::writableLocation(QStandardPaths::AppDataLocation))
                             .filePath(QStringLiteral("documents"));
    QDir().mkpath(path);
    return ::strdup(path.toUtf8().constData());
}

QStringList cleanCommand(const QStringList &value)
{
    QStringList command;
    for (const QString &part : value) {
        const QString clean = part.trimmed();
        if (!clean.isEmpty())
            command.append(clean);
    }
    return command;
}

}

UserPreferencesQt::UserPreferencesQt(ThemeManager *themes, QObject *parent)
    : QObject(parent),
      m_themes(themes)
{
    loadFromSettings();
    {
        QMutexLocker locker(&g_preferencesMutex);
        g_preferences = this;
        get_all_defaults_keys = getAllDefaultsKeysImpl;
        get_friendly_name = getFriendlyNameImpl;
        get_underlying_name = getUnderlyingNameImpl;
        get_user_default = getUserDefaultImpl;
        set_user_default = setUserDefaultImpl;
        remove_user_default = removeUserDefaultImpl;
        get_documents_directory = getDocumentsDirectoryImpl;
    }
    rebuildStyle();
}

UserPreferencesQt::~UserPreferencesQt()
{
    QMutexLocker locker(&g_preferencesMutex);
    if (g_preferences == this)
        g_preferences = nullptr;
    if (get_all_defaults_keys == getAllDefaultsKeysImpl)
        get_all_defaults_keys = nullptr;
    if (get_friendly_name == getFriendlyNameImpl)
        get_friendly_name = nullptr;
    if (get_underlying_name == getUnderlyingNameImpl)
        get_underlying_name = nullptr;
    if (get_user_default == getUserDefaultImpl)
        get_user_default = nullptr;
    if (set_user_default == setUserDefaultImpl)
        set_user_default = nullptr;
    if (remove_user_default == removeUserDefaultImpl)
        remove_user_default = nullptr;
    if (get_documents_directory == getDocumentsDirectoryImpl)
        get_documents_directory = nullptr;
}

void UserPreferencesQt::loadFromSettings()
{
    QSettings s = settings();
    m_themeName = s.value(QStringLiteral("ModernTheme"),
                          s.value(QStringLiteral("themeName"), QStringLiteral("Default"))).toString();
    m_fontFamily = s.value(QStringLiteral("Font Family"),
                           s.value(QStringLiteral("fontFamily"), QStringLiteral("Noto Sans Mono"))).toString();
    m_fontSize = std::clamp(s.value(QStringLiteral("Font Size"),
                                    s.value(QStringLiteral("fontSize"), 12)).toInt(), 6, 72);
    m_capsLockMapping = std::clamp(s.value(QStringLiteral("Caps Lock Mapping"), 1).toInt(), 0, 2);
    m_optionMapping = std::clamp(s.value(QStringLiteral("Option Mapping"), 0).toInt(), 0, 1);
    m_backtickMapEscape = s.value(QStringLiteral("Backtick Mapping Escape"), false).toBool();
    m_hideExtraKeysWithExternalKeyboard = s.value(QStringLiteral("Hide Extra Keys With External Keyboard"),
                                                  s.value(QStringLiteral("hideExtraKeysWithExternalKeyboard"), false)).toBool();
    m_overrideControlSpace = s.value(QStringLiteral("Override Control Space"), false).toBool();
    m_shouldDisableDimming = s.value(QStringLiteral("Disable Dimming"), false).toBool();
    m_hideStatusBar = s.value(QStringLiteral("Status Bar"), false).toBool();
    m_colorScheme = std::clamp(s.value(QStringLiteral("Color Scheme"), 0).toInt(), 0, 2);
    m_cursorStyle = std::clamp(s.value(QStringLiteral("Cursor Style"), 0).toInt(), 0, 2);
    m_blinkCursor = s.value(QStringLiteral("Blink Cursor"),
                            s.value(QStringLiteral("blinkCursor"), false)).toBool();
    m_bootCommand = cleanCommand(s.value(QStringLiteral("Boot Command"),
                                          s.value(QStringLiteral("bootCommand"), QStringList{QStringLiteral("/sbin/init")})).toStringList());
    m_launchCommand = cleanCommand(s.value(QStringLiteral("Init Command"),
                                            s.value(QStringLiteral("launchCommand"), QStringList{QStringLiteral("/bin/login"), QStringLiteral("-f"), QStringLiteral("root")})).toStringList());
    m_hostnameIsOverridden = s.contains(QStringLiteral("hostnameOverride"));
    m_hostnameOverride = s.value(QStringLiteral("hostnameOverride"), QSysInfo::machineHostName()).toString();
    if (m_bootCommand.isEmpty())
        m_bootCommand = {QStringLiteral("/sbin/init")};
}

void UserPreferencesQt::reload()
{
    const auto oldTheme = m_themeName;
    const auto oldFont = m_fontFamily;
    const int oldFontSize = m_fontSize;
    const int oldCaps = m_capsLockMapping;
    const int oldOption = m_optionMapping;
    const bool oldBacktick = m_backtickMapEscape;
    const bool oldHideKeys = m_hideExtraKeysWithExternalKeyboard;
    const bool oldControlSpace = m_overrideControlSpace;
    const bool oldDimming = m_shouldDisableDimming;
    const bool oldStatus = m_hideStatusBar;
    const int oldScheme = m_colorScheme;
    const int oldCursor = m_cursorStyle;
    const bool oldBlink = m_blinkCursor;
    const auto oldBoot = m_bootCommand;
    const auto oldLaunch = m_launchCommand;
    const auto oldHostname = m_hostnameOverride;
    const bool oldHostnameSet = m_hostnameIsOverridden;

    loadFromSettings();
    if (oldTheme != m_themeName) emit themeChanged();
    if (oldFont != m_fontFamily) emit fontFamilyChanged();
    if (oldFontSize != m_fontSize) emit fontSizeChanged();
    if (oldCaps != m_capsLockMapping || oldOption != m_optionMapping || oldBacktick != m_backtickMapEscape ||
        oldHideKeys != m_hideExtraKeysWithExternalKeyboard || oldControlSpace != m_overrideControlSpace)
        emit keyboardPreferencesChanged();
    if (oldDimming != m_shouldDisableDimming || oldStatus != m_hideStatusBar || oldScheme != m_colorScheme ||
        oldCursor != m_cursorStyle)
        emit appearancePreferencesChanged();
    if (oldBlink != m_blinkCursor) emit blinkCursorChanged();
    if (oldBoot != m_bootCommand || oldLaunch != m_launchCommand) emit commandChanged();
    if (oldHostname != m_hostnameOverride || oldHostnameSet != m_hostnameIsOverridden) emit hostnameChanged();
    rebuildStyle();
}

QString UserPreferencesQt::fontFamilyUserFacingName() const
{
    return m_fontFamily == QStringLiteral("ui-monospace") ? QStringLiteral("System") : m_fontFamily;
}

QString UserPreferencesQt::htermCursorShape() const
{
    switch (m_cursorStyle) {
    case 1: return QStringLiteral("BEAM");
    case 2: return QStringLiteral("UNDERLINE");
    default: return QStringLiteral("BLOCK");
    }
}

bool UserPreferencesQt::requestingDarkAppearance() const
{
    return m_colorScheme == 2;
}

void UserPreferencesQt::setStoredValue(const QString &key, const QVariant &value)
{
    QSettings s = settings();
    s.setValue(key, value);
    s.sync();
}

void UserPreferencesQt::save(const QString &key, const QVariant &value)
{
    setStoredValue(key, value);
}

void UserPreferencesQt::setThemeName(const QString &value)
{
    const QString name = value.trimmed().isEmpty() ? QStringLiteral("Default") : value.trimmed();
    if (m_themeName == name)
        return;
    m_themeName = name;
    save(QStringLiteral("ModernTheme"), m_themeName);
    emit themeChanged();
    rebuildStyle();
}

void UserPreferencesQt::setFontFamily(const QString &value)
{
    const QString family = value.trimmed().isEmpty() ? QStringLiteral("Noto Sans Mono") : value.trimmed();
    if (m_fontFamily == family)
        return;
    m_fontFamily = family;
    save(QStringLiteral("Font Family"), m_fontFamily);
    emit fontFamilyChanged();
    rebuildStyle();
}

void UserPreferencesQt::setFontSize(int value)
{
    const int size = std::clamp(value, 6, 72);
    if (m_fontSize == size)
        return;
    m_fontSize = size;
    save(QStringLiteral("Font Size"), m_fontSize);
    emit fontSizeChanged();
    rebuildStyle();
}

void UserPreferencesQt::setCapsLockMapping(int value)
{
    value = std::clamp(value, 0, 2);
    if (m_capsLockMapping == value) return;
    m_capsLockMapping = value;
    save(QStringLiteral("Caps Lock Mapping"), value);
    emit keyboardPreferencesChanged();
}

void UserPreferencesQt::setOptionMapping(int value)
{
    value = std::clamp(value, 0, 1);
    if (m_optionMapping == value) return;
    m_optionMapping = value;
    save(QStringLiteral("Option Mapping"), value);
    emit keyboardPreferencesChanged();
}

void UserPreferencesQt::setBacktickMapEscape(bool value)
{
    if (m_backtickMapEscape == value) return;
    m_backtickMapEscape = value;
    save(QStringLiteral("Backtick Mapping Escape"), value);
    emit keyboardPreferencesChanged();
}

void UserPreferencesQt::setHideExtraKeysWithExternalKeyboard(bool value)
{
    if (m_hideExtraKeysWithExternalKeyboard == value) return;
    m_hideExtraKeysWithExternalKeyboard = value;
    save(QStringLiteral("Hide Extra Keys With External Keyboard"), value);
    emit keyboardPreferencesChanged();
}

void UserPreferencesQt::setOverrideControlSpace(bool value)
{
    if (m_overrideControlSpace == value) return;
    m_overrideControlSpace = value;
    save(QStringLiteral("Override Control Space"), value);
    emit keyboardPreferencesChanged();
}

void UserPreferencesQt::setShouldDisableDimming(bool value)
{
    if (m_shouldDisableDimming == value) return;
    m_shouldDisableDimming = value;
    save(QStringLiteral("Disable Dimming"), value);
    emit appearancePreferencesChanged();
}

void UserPreferencesQt::setHideStatusBar(bool value)
{
    if (m_hideStatusBar == value) return;
    m_hideStatusBar = value;
    save(QStringLiteral("Status Bar"), value);
    emit appearancePreferencesChanged();
}

void UserPreferencesQt::setColorScheme(int value)
{
    value = std::clamp(value, 0, 2);
    if (m_colorScheme == value) return;
    m_colorScheme = value;
    save(QStringLiteral("Color Scheme"), value);
    emit appearancePreferencesChanged();
    rebuildStyle();
}

void UserPreferencesQt::setCursorStyle(int value)
{
    value = std::clamp(value, 0, 2);
    if (m_cursorStyle == value) return;
    m_cursorStyle = value;
    save(QStringLiteral("Cursor Style"), value);
    emit appearancePreferencesChanged();
    rebuildStyle();
}

void UserPreferencesQt::setBlinkCursor(bool value)
{
    if (m_blinkCursor == value) return;
    m_blinkCursor = value;
    save(QStringLiteral("Blink Cursor"), value);
    emit blinkCursorChanged();
    rebuildStyle();
}

void UserPreferencesQt::setBootCommand(const QStringList &value)
{
    QStringList command = cleanCommand(value);
    if (command.isEmpty()) command = {QStringLiteral("/sbin/init")};
    if (m_bootCommand == command) return;
    m_bootCommand = command;
    save(QStringLiteral("Boot Command"), m_bootCommand);
    emit commandChanged();
}

void UserPreferencesQt::setLaunchCommand(const QStringList &value)
{
    const QStringList command = cleanCommand(value);
    if (m_launchCommand == command) return;
    m_launchCommand = command;
    save(QStringLiteral("Init Command"), m_launchCommand);
    emit commandChanged();
}

void UserPreferencesQt::setHostnameOverride(const QString &value)
{
    const QString hostname = value.trimmed();
    if (m_hostnameOverride == hostname && m_hostnameIsOverridden == !hostname.isEmpty())
        return;
    m_hostnameOverride = hostname.isEmpty() ? QSysInfo::machineHostName() : hostname;
    m_hostnameIsOverridden = !hostname.isEmpty();
    if (m_hostnameIsOverridden)
        save(QStringLiteral("hostnameOverride"), m_hostnameOverride);
    else {
        QSettings s = settings();
        s.remove(QStringLiteral("hostnameOverride"));
        s.sync();
    }
    emit hostnameChanged();
}

QStringList UserPreferencesQt::encodeCommand(const QStringList &command) const
{
    return cleanCommand(command);
}

void UserPreferencesQt::rebuildStyle()
{
    if (m_themes)
        m_terminalStyle = m_themes->styleForName(m_themeName);
    else
        m_terminalStyle = {
            {QStringLiteral("backgroundColor"), QStringLiteral("#000000")},
            {QStringLiteral("foregroundColor"), QStringLiteral("#f5f5f7")},
            {QStringLiteral("cursorColor"), QStringLiteral("#ffffff")},
            {QStringLiteral("selectionColor"), QStringLiteral("#264f78")}
        };
    m_terminalStyle.insert(QStringLiteral("fontFamily"), m_fontFamily);
    m_terminalStyle.insert(QStringLiteral("fontSize"), m_fontSize);
    m_terminalStyle.insert(QStringLiteral("blinkCursor"), m_blinkCursor);
    m_terminalStyle.insert(QStringLiteral("cursorShape"), htermCursorShape().toLower());
    m_terminalStyle.insert(QStringLiteral("colorScheme"), m_colorScheme);
    emit styleChanged();
}
