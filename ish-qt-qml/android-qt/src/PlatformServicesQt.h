#pragma once

#include <QObject>
#include <QString>

class PlatformServicesQt final : public QObject
{
    Q_OBJECT
public:
    explicit PlatformServicesQt(QObject *parent = nullptr);

    Q_INVOKABLE QString pasteText() const;
    Q_INVOKABLE void reportError(const QString &title, const QString &message);

signals:
    void errorReported(const QString &title, const QString &message);
};
