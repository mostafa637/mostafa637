#pragma once

#include <QObject>
#include <QString>

class PlatformServicesQt final : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QString clipboardText READ clipboardText NOTIFY clipboardChanged)

public:
    explicit PlatformServicesQt(QObject *parent = nullptr);

    Q_INVOKABLE QString pasteText() const;
    QString clipboardText() const;

    Q_INVOKABLE void copyText(const QString &text);
    Q_INVOKABLE void clearClipboard();
    Q_INVOKABLE bool hasClipboardText() const;
    Q_INVOKABLE void reportError(const QString &title, const QString &message);

signals:
    void clipboardChanged();
    void errorReported(const QString &title, const QString &message);
};
