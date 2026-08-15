#include "UserPreferencesQt.h"
#include "ThemeManager.h"

#include <QSettings>
#include <algorithm>

namespace {
QSettings settings()
{
    return QSettings(QStringLiteral("iSH"), QStringLiteral("iSH Qt"));
}
}

UserPreferencesQt::UserPreferencesQt(ThemeManager *themes, QObject *parent)
    : QObject(parent),
      m_themes(themes)
{
    QSettings s = settings();
    m_themeName = s.value(QStringLiteral("themeName"), QStringLiteral("Default")).toString();
    m_fontFamily = s.value(QStringLiteral("fontFamily"), QStringLiteral("Noto Sans Mono")).toString();
    m_fontSize = std::clamp(s.value(QStringLiteral("fontSize"), 14).toInt(), 8, 36);
    m_blinkCursor = s.value(QStringLiteral("blinkCursor"), true).toBool();
    m_bootCommand = s.value(QStringLiteral("bootCommand"), QStringList{QStringLiteral("/bin/sh")}).toStringList();
    m_launchCommand = s.value(QStringLiteral("launchCommand"), QStringList{}).toStringList();
    m_hideExtraKeysWithExternalKeyboard = s.value(QStringLiteral("hideExtraKeysWithExternalKeyboard"), false).toBool();
    if (m_bootCommand.isEmpty())
        m_bootCommand = {QStringLiteral("/bin/sh")};
    rebuildStyle();
}

void UserPreferencesQt::setThemeName(const QString &value)
{
    const QString name = value.trimmed().isEmpty() ? QStringLiteral("Default") : value.trimmed();
    if (m_themeName == name)
        return;
    m_themeName = name;
    save(QStringLiteral("themeName"), m_themeName);
    rebuildStyle();
}

void UserPreferencesQt::setFontFamily(const QString &value)
{
    const QString family = value.trimmed().isEmpty() ? QStringLiteral("Noto Sans Mono") : value.trimmed();
    if (m_fontFamily == family)
        return;
    m_fontFamily = family;
    save(QStringLiteral("fontFamily"), m_fontFamily);
    emit fontFamilyChanged();
}

void UserPreferencesQt::setFontSize(int value)
{
    const int size = std::clamp(value, 8, 36);
    if (m_fontSize == size)
        return;
    m_fontSize = size;
    save(QStringLiteral("fontSize"), m_fontSize);
    emit fontSizeChanged();
}

void UserPreferencesQt::setBlinkCursor(bool value)
{
    if (m_blinkCursor == value)
        return;
    m_blinkCursor = value;
    save(QStringLiteral("blinkCursor"), m_blinkCursor);
    emit blinkCursorChanged();
}

void UserPreferencesQt::setBootCommand(const QStringList &value)
{
    QStringList command;
    for (const QString &part : value) {
        const QString clean = part.trimmed();
        if (!clean.isEmpty())
            command.append(clean);
    }
    if (command.isEmpty())
        command = {QStringLiteral("/bin/sh")};
    if (m_bootCommand == command)
        return;
    m_bootCommand = command;
    save(QStringLiteral("bootCommand"), m_bootCommand);
    emit commandChanged();
}

void UserPreferencesQt::setLaunchCommand(const QStringList &value)
{
    QStringList command;
    for (const QString &part : value) {
        const QString clean = part.trimmed();
        if (!clean.isEmpty())
            command.append(clean);
    }
    if (m_launchCommand == command)
        return;
    m_launchCommand = command;
    save(QStringLiteral("launchCommand"), m_launchCommand);
    emit commandChanged();
}

void UserPreferencesQt::setHideExtraKeysWithExternalKeyboard(bool value)
{
    if (m_hideExtraKeysWithExternalKeyboard == value)
        return;
    m_hideExtraKeysWithExternalKeyboard = value;
    save(QStringLiteral("hideExtraKeysWithExternalKeyboard"), m_hideExtraKeysWithExternalKeyboard);
    emit hideExtraKeysWithExternalKeyboardChanged();
}

QStringList UserPreferencesQt::encodeCommand(const QStringList &command) const
{
    QStringList encoded;
    for (const QString &part : command) {
        const QString clean = part.trimmed();
        if (!clean.isEmpty())
            encoded.append(clean);
    }
    return encoded;
}

void UserPreferencesQt::save(const QString &key, const QVariant &value)
{
    QSettings s = settings();
    s.setValue(key, value);
    s.sync();
}

void UserPreferencesQt::rebuildStyle()
{
    if (m_themes)
        m_terminalStyle = m_themes->styleForName(m_themeName);
    else
        m_terminalStyle = {
            {QStringLiteral("backgroundColor"), QStringLiteral("#000000")},
            {QStringLiteral("foregroundColor"), QStringLiteral("#f5f5f7")},
            {QStringLiteral("cursorColor"), QStringLiteral("#0a84ff")},
            {QStringLiteral("selectionColor"), QStringLiteral("#264f78")}
        };
    m_terminalStyle.insert(QStringLiteral("fontFamily"), m_fontFamily);
    m_terminalStyle.insert(QStringLiteral("fontSize"), m_fontSize);
    m_terminalStyle.insert(QStringLiteral("blinkCursor"), m_blinkCursor);
    emit styleChanged();
}
