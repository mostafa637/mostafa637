#pragma once

#include <QObject>
#include <QPointer>
#include <QWebSocket>


class WebSocketTransport final : public QObject
{
    Q_OBJECT
public:
    explicit WebSocketTransport(QWebSocket *socket, QObject *parent = nullptr);

    QWebSocket *socket() const { return m_socket; }
    void sendText(const QString &text);
    void sendBinary(const QByteArray &data);

signals:
    void textReceived(const QString &text);
    void binaryReceived(const QByteArray &data);
    void disconnected();

private:
    QPointer<QWebSocket> m_socket;
};
