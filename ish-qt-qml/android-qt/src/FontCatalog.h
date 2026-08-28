#pragma once

#include <QObject>
#include <QStringList>

class FontCatalog final : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QStringList families READ families NOTIFY familiesChanged)

public:
    explicit FontCatalog(QObject *parent = nullptr);

    QStringList families() const { return m_families; }
    Q_INVOKABLE bool contains(const QString &family) const;

signals:
    void familiesChanged();

private:
    QStringList m_families;
};
