#include "FontCatalog.h"

#include <QFontDatabase>

FontCatalog::FontCatalog(QObject *parent)
    : QObject(parent),
      m_families(QFontDatabase().families())
{
    if (!m_families.contains(QStringLiteral("Noto Sans Mono"), Qt::CaseInsensitive))
        m_families.prepend(QStringLiteral("Noto Sans Mono"));
    if (!m_families.contains(QStringLiteral("Monospace"), Qt::CaseInsensitive))
        m_families.append(QStringLiteral("Monospace"));
    m_families.removeDuplicates();
}

bool FontCatalog::contains(const QString &family) const
{
    return m_families.contains(family, Qt::CaseInsensitive);
}
