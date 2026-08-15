#include "ThemeManager.h"

ThemeManager::ThemeManager(QObject *parent)
    : QObject(parent),
      m_themeNames{QStringLiteral("Default"), QStringLiteral("Light"), QStringLiteral("Solarized Dark")}
{
}

QVariantMap ThemeManager::styleForName(const QString &name) const
{
    if (name == QStringLiteral("Light")) {
        return {
            {QStringLiteral("backgroundColor"), QStringLiteral("#f2f2f7")},
            {QStringLiteral("foregroundColor"), QStringLiteral("#1c1c1e")},
            {QStringLiteral("cursorColor"), QStringLiteral("#007aff")},
            {QStringLiteral("selectionColor"), QStringLiteral("#b7d7ff")}
        };
    }
    if (name == QStringLiteral("Solarized Dark")) {
        return {
            {QStringLiteral("backgroundColor"), QStringLiteral("#002b36")},
            {QStringLiteral("foregroundColor"), QStringLiteral("#839496")},
            {QStringLiteral("cursorColor"), QStringLiteral("#b58900")},
            {QStringLiteral("selectionColor"), QStringLiteral("#335b63")}
        };
    }
    return {
        {QStringLiteral("backgroundColor"), QStringLiteral("#000000")},
        {QStringLiteral("foregroundColor"), QStringLiteral("#f5f5f7")},
        {QStringLiteral("cursorColor"), QStringLiteral("#0a84ff")},
        {QStringLiteral("selectionColor"), QStringLiteral("#264f78")}
    };
}
