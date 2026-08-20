#pragma once

#include <QObject>
#include <QStringList>
#include <QVariantMap>

class ThemeManager;

class UserPreferencesQt final : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QVariantMap terminalStyle READ terminalStyle NOTIFY styleChanged)
    Q_PROPERTY(QString themeName READ themeName WRITE setThemeName NOTIFY themeChanged)
    Q_PROPERTY(QString fontFamily READ fontFamily WRITE setFontFamily NOTIFY fontFamilyChanged)
    Q_PROPERTY(QString fontFamilyUserFacingName READ fontFamilyUserFacingName NOTIFY fontFamilyChanged)
    Q_PROPERTY(int fontSize READ fontSize WRITE setFontSize NOTIFY fontSizeChanged)
    Q_PROPERTY(int capsLockMapping READ capsLockMapping WRITE setCapsLockMapping NOTIFY keyboardPreferencesChanged)
    Q_PROPERTY(int optionMapping READ optionMapping WRITE setOptionMapping NOTIFY keyboardPreferencesChanged)
    Q_PROPERTY(bool backtickMapEscape READ backtickMapEscape WRITE setBacktickMapEscape NOTIFY keyboardPreferencesChanged)
    Q_PROPERTY(bool hideExtraKeysWithExternalKeyboard READ hideExtraKeysWithExternalKeyboard WRITE setHideExtraKeysWithExternalKeyboard NOTIFY keyboardPreferencesChanged)
    Q_PROPERTY(bool overrideControlSpace READ overrideControlSpace WRITE setOverrideControlSpace NOTIFY keyboardPreferencesChanged)
    Q_PROPERTY(bool shouldDisableDimming READ shouldDisableDimming WRITE setShouldDisableDimming NOTIFY appearancePreferencesChanged)
    Q_PROPERTY(bool hideStatusBar READ hideStatusBar WRITE setHideStatusBar NOTIFY appearancePreferencesChanged)
    Q_PROPERTY(int colorScheme READ colorScheme WRITE setColorScheme NOTIFY appearancePreferencesChanged)
    Q_PROPERTY(int cursorStyle READ cursorStyle WRITE setCursorStyle NOTIFY appearancePreferencesChanged)
    Q_PROPERTY(QString htermCursorShape READ htermCursorShape NOTIFY appearancePreferencesChanged)
    Q_PROPERTY(bool requestingDarkAppearance READ requestingDarkAppearance NOTIFY appearancePreferencesChanged)
    Q_PROPERTY(bool blinkCursor READ blinkCursor WRITE setBlinkCursor NOTIFY blinkCursorChanged)
    Q_PROPERTY(QStringList bootCommand READ bootCommand WRITE setBootCommand NOTIFY commandChanged)
    Q_PROPERTY(QStringList launchCommand READ launchCommand WRITE setLaunchCommand NOTIFY commandChanged)
    Q_PROPERTY(QString hostnameOverride READ hostnameOverride WRITE setHostnameOverride NOTIFY hostnameChanged)
    Q_PROPERTY(bool hostnameIsOverridden READ hostnameIsOverridden NOTIFY hostnameChanged)

public:
    explicit UserPreferencesQt(ThemeManager *themes, QObject *parent = nullptr);
    ~UserPreferencesQt() override;

    QVariantMap terminalStyle() const { return m_terminalStyle; }
    QString themeName() const { return m_themeName; }
    QString fontFamily() const { return m_fontFamily; }
    QString fontFamilyUserFacingName() const;
    int fontSize() const { return m_fontSize; }
    int capsLockMapping() const { return m_capsLockMapping; }
    int optionMapping() const { return m_optionMapping; }
    bool backtickMapEscape() const { return m_backtickMapEscape; }
    bool hideExtraKeysWithExternalKeyboard() const { return m_hideExtraKeysWithExternalKeyboard; }
    bool overrideControlSpace() const { return m_overrideControlSpace; }
    bool shouldDisableDimming() const { return m_shouldDisableDimming; }
    bool hideStatusBar() const { return m_hideStatusBar; }
    int colorScheme() const { return m_colorScheme; }
    int cursorStyle() const { return m_cursorStyle; }
    QString htermCursorShape() const;
    bool requestingDarkAppearance() const;
    bool blinkCursor() const { return m_blinkCursor; }
    QStringList bootCommand() const { return m_bootCommand; }
    QStringList launchCommand() const { return m_launchCommand; }
    QString hostnameOverride() const { return m_hostnameOverride; }
    bool hostnameIsOverridden() const { return m_hostnameIsOverridden; }

    void setThemeName(const QString &value);
    void setFontFamily(const QString &value);
    void setFontSize(int value);
    void setCapsLockMapping(int value);
    void setOptionMapping(int value);
    void setBacktickMapEscape(bool value);
    void setHideExtraKeysWithExternalKeyboard(bool value);
    void setOverrideControlSpace(bool value);
    void setShouldDisableDimming(bool value);
    void setHideStatusBar(bool value);
    void setColorScheme(int value);
    void setCursorStyle(int value);
    void setBlinkCursor(bool value);
    void setBootCommand(const QStringList &value);
    void setLaunchCommand(const QStringList &value);
    void setHostnameOverride(const QString &value);

    Q_INVOKABLE QStringList encodeCommand(const QStringList &command) const;
    Q_INVOKABLE void reload();

signals:
    void styleChanged();
    void themeChanged();
    void fontFamilyChanged();
    void fontSizeChanged();
    void keyboardPreferencesChanged();
    void appearancePreferencesChanged();
    void blinkCursorChanged();
    void commandChanged();
    void hostnameChanged();

private:
    void loadFromSettings();
    void save(const QString &key, const QVariant &value);
    void rebuildStyle();
    void setStoredValue(const QString &key, const QVariant &value);

    ThemeManager *m_themes = nullptr;
    QVariantMap m_terminalStyle;
    QString m_themeName;
    QString m_fontFamily;
    int m_fontSize = 12;
    int m_capsLockMapping = 1;
    int m_optionMapping = 0;
    bool m_backtickMapEscape = false;
    bool m_hideExtraKeysWithExternalKeyboard = false;
    bool m_overrideControlSpace = false;
    bool m_shouldDisableDimming = false;
    bool m_hideStatusBar = false;
    int m_colorScheme = 0;
    int m_cursorStyle = 0;
    bool m_blinkCursor = false;
    QStringList m_bootCommand;
    QStringList m_launchCommand;
    QString m_hostnameOverride;
    bool m_hostnameIsOverridden = false;
};
