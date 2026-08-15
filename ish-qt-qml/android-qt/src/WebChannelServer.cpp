#include "WebChannelServer.h"
#include "WebSocketTransport.h"

#include <QHostAddress>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonValue>
#include <QWebSocket>
#include <QWebSocketServer>
#include <utility>

WebChannelServer::WebChannelServer(QObject *parent)
    : QObject(parent),
      m_server(new QWebSocketServer(QStringLiteral("iSH Qt terminal"),
                                    QWebSocketServer::NonSecureMode, this))
{
    connect(m_server, &QWebSocketServer::newConnection,
            this, &WebChannelServer::acceptConnection);
}

WebChannelServer::~WebChannelServer()
{
    stop();
}

bool WebChannelServer::start()
{
    if (m_server->isListening())
        return true;

    if (!m_server->listen(QHostAddress::LocalHost, 0)) {
        emit serverError(m_server->errorString());
        return false;
    }

    m_url = QStringLiteral("ws://127.0.0.1:%1").arg(m_server->serverPort());
    emit urlChanged();
    return true;
}

void WebChannelServer::stop()
{
    for (WebSocketTransport *transport : std::as_const(m_transports)) {
        if (transport && transport->socket())
            transport->socket()->close();
        if (transport)
            transport->deleteLater();
    }
    m_transports.clear();
    if (m_server->isListening())
        m_server->close();
    if (!m_url.isEmpty()) {
        m_url.clear();
        emit urlChanged();
    }
}

void WebChannelServer::sendOutput(const QString &value)
{
    sendOutputBytes(value.toUtf8());
}

void WebChannelServer::sendOutputBytes(const QByteArray &value)
{
    for (WebSocketTransport *transport : std::as_const(m_transports)) {
        if (transport && transport->socket())
            transport->sendBinary(value);
    }
}

void WebChannelServer::acceptConnection()
{
    while (m_server->hasPendingConnections()) {
        QWebSocket *socket = m_server->nextPendingConnection();
        auto *transport = new WebSocketTransport(socket, this);
        m_transports.append(transport);
        connect(transport, &WebSocketTransport::textReceived,
                this, &WebChannelServer::handleText);
        connect(transport, &WebSocketTransport::binaryReceived,
                this, [this](const QByteArray &data) {
                    emit inputReceived(QString::fromUtf8(data));
                });
        connect(transport, &WebSocketTransport::disconnected,
                this, &WebChannelServer::removeTransport);
    }
}

void WebChannelServer::handleText(const QString &text)
{
    const QJsonDocument document = QJsonDocument::fromJson(text.toUtf8());
    if (document.isObject()) {
        const QJsonObject object = document.object();
        const QJsonValue value = object.value(QStringLiteral("input"));
        if (value.isString()) {
            emit inputReceived(value.toString());
            return;
        }
        const QJsonValue data = object.value(QStringLiteral("data"));
        if (data.isString()) {
            emit inputReceived(data.toString());
            return;
        }
    }
    emit inputReceived(text);
}

void WebChannelServer::removeTransport()
{
    auto *transport = qobject_cast<WebSocketTransport *>(sender());
    if (!transport)
        return;
    m_transports.removeAll(transport);
    transport->deleteLater();
}
