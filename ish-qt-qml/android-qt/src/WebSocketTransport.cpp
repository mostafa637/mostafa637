#include "WebSocketTransport.h"

#include <QWebSocket>
#include <utility>

WebSocketTransport::WebSocketTransport(QWebSocket *socket, QObject *parent)
    : QObject(parent),
      m_socket(socket)
{
    if (!m_socket)
        return;
    m_socket->setParent(this);
    connect(m_socket, &QWebSocket::textMessageReceived,
            this, &WebSocketTransport::textReceived);
    connect(m_socket, &QWebSocket::binaryMessageReceived,
            this, &WebSocketTransport::binaryReceived);
    connect(m_socket, &QWebSocket::disconnected,
            this, &WebSocketTransport::disconnected);
}

void WebSocketTransport::sendText(const QString &text)
{
    if (m_socket)
        m_socket->sendTextMessage(text);
}

void WebSocketTransport::sendBinary(const QByteArray &data)
{
    if (m_socket)
        m_socket->sendBinaryMessage(data);
}
