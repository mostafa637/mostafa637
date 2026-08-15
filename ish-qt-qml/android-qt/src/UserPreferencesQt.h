#pragma once

#include <QObject>
#include <QStringList>
#include <QVariantMap>

class ThemeManager;

class UserPreferencesQt final : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QVariantMap terminalStyle READ terminalStyle NOTIFY styleChanged)
    Q_PROPERTY(QString themeName READ themeName WRITE setThemeName NOTIFY styleChanged)
    Q_PROPERTY(QString fontFamily READ fontFamily WRITE setFontFamily NOTIFY fontFamilyChanged)
    Q_PROPERTY(int fontSize READ fontSize WRITE setFontSize NOTIFY fontSizeChanged)
    Q_PROPERTY(bool blinkCursor READ blinkCursor WRITE setBlinkCursor NOTIFY blinkCursorChanged)
    Q_PROPERTY(QStringList bootCommand READ bootCommand WRITE setBootCommand NOTIFY commandChanged)
    Q_PROPERTY(QStringList launchCommand READ launchCommand WRITE setLaunchCommand NOTIFY commandChanged)
    Q_PROPERTY(bool hideExtraKeysWithExternalKeyboard READ hideExtraKeysWithExternalKeyboard WRITE setHideExtraKeysWithExternalKeyboard NOTIFY hideExtraKeysWithExternalKeyboardChanged)

public:
    explicit UserPreferencesQt(ThemeManager *themes, QObject *parent = nullptr);

    QVariantMap terminalStyle() const { return m_terminalStyle; }
    QString themeName() const { return m_themeName; }
    QString fontFamily() const { return m_fontFamily; }
    int fontSize() const { return m_fontSize; }
    bool blinkCursor() const { return m_blinkCursor; }
    QStringList bootCommand() const { return m_bootCommand; }
    QStringList launchCommand() const { return m_launchCommand; }
    bool hideExtraKeysWithExternalKeyboard() const { return m_hideExtraKeysWithExternalKeyboard; }

    void setThemeName(const QString &value);
    void setFontFamily(const QString &value);
    void setFontSize(int value);
    void setBlinkCursor(bool value);
    void setBootCommand(const QStringList &value);
    void setLaunchCommand(const QStringList &value);
    void setHideExtraKeysWithExternalKeyboard(bool value);

    Q_INVOKABLE QStringList encodeCommand(const QStringList &command) const;

signals:
    void styleChanged();
    void fontFamilyChanged();
    void fontSizeChanged();
    void blinkCursorChanged();
    void commandChanged();
    void hideExtraKeysWithExternalKeyboardChanged();

private:
    void save(const QString &key, const QVariant &value);
    void rebuildStyle();

    ThemeManager *m_themes = nullptr;
    QVariantMap m_terminalStyle;
    QString m_themeName;
    QString m_fontFamily;
    int m_fontSize = 14;
    bool m_blinkCursor = true;
    QStringList m_bootCommand;
    QStringList m_launchCommand;
    bool m_hideExtraKeysWithExternalKeyboard = false;
};
