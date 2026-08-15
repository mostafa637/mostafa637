#pragma once

#include <QObject>
#include <QStringList>
#include <QVariantMap>

class ThemeManager final : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QStringList themeNames READ themeNames NOTIFY themeNamesChanged)

public:
    explicit ThemeManager(QObject *parent = nullptr);

    QStringList themeNames() const { return m_themeNames; }
    Q_INVOKABLE QVariantMap styleForName(const QString &name) const;

signals:
    void themeNamesChanged();
    void themeError(const QString &message);

private:
    QStringList m_themeNames;
};
