#pragma once

#include <QObject>
#include <QString>

class QFile;

class PlatformServicesQt final : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QString clipboardText READ clipboardText NOTIFY clipboardChanged)
    Q_PROPERTY(QString diagnosticLogPath READ diagnosticLogPath CONSTANT)

public:
    explicit PlatformServicesQt(QObject *parent = nullptr);

    Q_INVOKABLE QString pasteText() const;
    QString clipboardText() const;

    Q_INVOKABLE void copyText(const QString &text);
    Q_INVOKABLE void clearClipboard();
    Q_INVOKABLE bool hasClipboardText() const;
    Q_INVOKABLE void reportError(const QString &title, const QString &message);
    QString diagnosticLogPath() const { return m_diagnosticLogPath; }
    Q_INVOKABLE void logDiagnostic(const QString &title, const QString &message);
    void installCrashHandler();

signals:
    void clipboardChanged();
    void errorReported(const QString &title, const QString &message);

private:
    void appendDiagnostic(const QString &line);

    QString m_diagnosticLogPath;
    QFile *m_diagnosticLog = nullptr;
};
