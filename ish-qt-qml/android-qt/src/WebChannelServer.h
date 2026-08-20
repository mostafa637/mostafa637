#pragma once

#include <QObject>
#include <QString>
#include <QHash>
#include <QList>

class QTcpServer;
class QTcpSocket;
class QWebSocketServer;
class QWebSocket;
class WebSocketTransport;

class WebChannelServer final : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QString url READ url NOTIFY urlChanged)
    Q_PROPERTY(QString pageUrl READ pageUrl NOTIFY pageUrlChanged)

public:
    explicit WebChannelServer(QObject *parent = nullptr);
    ~WebChannelServer() override;

    QString url() const { return m_url; }
    QString pageUrl() const { return m_pageUrl; }
    Q_INVOKABLE bool start();
    Q_INVOKABLE void stop();

public slots:
    void sendOutput(const QString &value);
    void sendOutputBytes(const QByteArray &value);

signals:
    void urlChanged();
    void pageUrlChanged();
    void inputReceived(const QString &value);
    void serverError(const QString &message);

private slots:
    void acceptConnection();
    void acceptHttpConnection();
    void handleHttpRequest();
    void handleText(const QString &text);
    void removeTransport();
    void removeHttpSocket();

private:
    void sendHttpResponse(QTcpSocket *socket, int status, const QByteArray &reason,
                          const QByteArray &contentType, const QByteArray &body);

    QWebSocketServer *m_server = nullptr;
    QTcpServer *m_httpServer = nullptr;
    QList<WebSocketTransport *> m_transports;
    QHash<QTcpSocket *, QByteArray> m_httpRequests;
    QByteArray m_pendingOutput;
    QString m_url;
    QString m_pageUrl;
};
