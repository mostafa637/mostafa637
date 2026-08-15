#pragma once

#include <QObject>
#include <QString>
#include <QList>

class QWebSocketServer;
class QWebSocket;
class WebSocketTransport;

class WebChannelServer final : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QString url READ url NOTIFY urlChanged)

public:
    explicit WebChannelServer(QObject *parent = nullptr);
    ~WebChannelServer() override;

    QString url() const { return m_url; }
    Q_INVOKABLE bool start();
    Q_INVOKABLE void stop();

public slots:
    void sendOutput(const QString &value);
    void sendOutputBytes(const QByteArray &value);

signals:
    void urlChanged();
    void inputReceived(const QString &value);
    void serverError(const QString &message);

private slots:
    void acceptConnection();
    void handleText(const QString &text);
    void removeTransport();

private:
    QWebSocketServer *m_server = nullptr;
    QList<WebSocketTransport *> m_transports;
    QString m_url;
};
