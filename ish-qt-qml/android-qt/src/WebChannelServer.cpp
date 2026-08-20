#include "WebChannelServer.h"
#include "WebSocketTransport.h"

#include <QFile>
#include <QHostAddress>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonValue>
#include <QRegularExpression>
#include <QTcpServer>
#include <QTcpSocket>
#include <QUrl>
#include <QWebSocket>
#include <QWebSocketServer>
#include <utility>

namespace {

QByteArray contentTypeFor(const QString &path)
{
    if (path.endsWith(QStringLiteral(".html")))
        return QByteArrayLiteral("text/html; charset=utf-8");
    if (path.endsWith(QStringLiteral(".css")))
        return QByteArrayLiteral("text/css; charset=utf-8");
    if (path.endsWith(QStringLiteral(".js")))
        return QByteArrayLiteral("application/javascript; charset=utf-8");
    return QByteArrayLiteral("application/octet-stream");
}

bool isTerminalAsset(const QString &path)
{
    return path == QStringLiteral("/terminal/term.html") ||
           path == QStringLiteral("/terminal/term.css") ||
           path == QStringLiteral("/terminal/term.js") ||
           path == QStringLiteral("/terminal/qwebchannel.js");
}

} // namespace

WebChannelServer::WebChannelServer(QObject *parent)
    : QObject(parent),
      m_server(new QWebSocketServer(QStringLiteral("iSH Qt terminal"),
                                    QWebSocketServer::NonSecureMode, this)),
      m_httpServer(new QTcpServer(this))
{
    connect(m_server, &QWebSocketServer::newConnection,
            this, &WebChannelServer::acceptConnection);
    connect(m_httpServer, &QTcpServer::newConnection,
            this, &WebChannelServer::acceptHttpConnection);
}

WebChannelServer::~WebChannelServer()
{
    stop();
}

bool WebChannelServer::start()
{
    if (m_server->isListening() && m_httpServer->isListening())
        return true;

    if (m_server->isListening())
        m_server->close();
    if (m_httpServer->isListening())
        m_httpServer->close();

    if (!m_server->listen(QHostAddress::LocalHost, 0)) {
        qWarning() << "[ish-qt] WebChannelServer: ws listen failed:" << m_server->errorString();
        emit serverError(m_server->errorString());
        return false;
    }
    if (!m_httpServer->listen(QHostAddress::LocalHost, 0)) {
        const QString error = m_httpServer->errorString();
        qWarning() << "[ish-qt] WebChannelServer: http listen failed:" << error;
        m_server->close();
        emit serverError(error);
        return false;
    }

    m_url = QStringLiteral("ws://127.0.0.1:%1").arg(m_server->serverPort());
    m_pageUrl = QStringLiteral("http://127.0.0.1:%1/terminal/term.html")
                    .arg(m_httpServer->serverPort());
    qWarning() << "[ish-qt] WebChannelServer listening: ws=" << m_url << "page=" << m_pageUrl;
    emit urlChanged();
    emit pageUrlChanged();
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
    m_pendingOutput.clear();

    const auto sockets = m_httpRequests.keys();
    for (QTcpSocket *socket : sockets) {
        if (socket)
            socket->close();
        if (socket)
            socket->deleteLater();
    }
    m_httpRequests.clear();

    if (m_server->isListening())
        m_server->close();
    if (m_httpServer->isListening())
        m_httpServer->close();
    if (!m_url.isEmpty()) {
        m_url.clear();
        emit urlChanged();
    }
    if (!m_pageUrl.isEmpty()) {
        m_pageUrl.clear();
        emit pageUrlChanged();
    }
}

void WebChannelServer::sendOutput(const QString &value)
{
    sendOutputBytes(value.toUtf8());
}

void WebChannelServer::sendOutputBytes(const QByteArray &value)
{
    if (value.isEmpty())
        return;

    if (m_transports.isEmpty()) {
        constexpr qsizetype maxPendingOutput = 1024 * 1024;
        if (m_pendingOutput.size() + value.size() > maxPendingOutput) {
            const qsizetype keep = qMax<qsizetype>(0, maxPendingOutput - value.size());
            if (keep == 0)
                m_pendingOutput.clear();
            else
                m_pendingOutput = m_pendingOutput.right(keep);
        }
        m_pendingOutput += value;
        return;
    }

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
        if (!m_pendingOutput.isEmpty()) {
            transport->sendBinary(m_pendingOutput);
            m_pendingOutput.clear();
        }
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

void WebChannelServer::acceptHttpConnection()
{
    while (m_httpServer->hasPendingConnections()) {
        QTcpSocket *socket = m_httpServer->nextPendingConnection();
        m_httpRequests.insert(socket, QByteArray());
        connect(socket, &QTcpSocket::readyRead,
                this, &WebChannelServer::handleHttpRequest);
        connect(socket, &QTcpSocket::disconnected,
                this, &WebChannelServer::removeHttpSocket);
    }
}

void WebChannelServer::handleHttpRequest()
{
    auto *socket = qobject_cast<QTcpSocket *>(sender());
    if (!socket || !m_httpRequests.contains(socket))
        return;

    QByteArray &request = m_httpRequests[socket];
    request += socket->readAll();
    if (!request.contains("\r\n\r\n"))
        return;

    const QByteArray requestLine = request.left(request.indexOf("\r\n"));
    const QList<QByteArray> fields = requestLine.split(' ');
    if (fields.size() < 2 || fields.at(0) != QByteArrayLiteral("GET")) {
        sendHttpResponse(socket, 405, QByteArrayLiteral("Method Not Allowed"),
                         QByteArrayLiteral("text/plain; charset=utf-8"),
                         QByteArrayLiteral("Method Not Allowed\n"));
        return;
    }

    const QUrl requestedUrl(QString::fromUtf8(fields.at(1)));
    const QString path = requestedUrl.path();
    if (!isTerminalAsset(path) || path.contains(QStringLiteral(".."))) {
        sendHttpResponse(socket, 404, QByteArrayLiteral("Not Found"),
                         QByteArrayLiteral("text/plain; charset=utf-8"),
                         QByteArrayLiteral("Not Found\n"));
        return;
    }

    QFile resource(QStringLiteral(":/ish-assets") + path);
    if (!resource.open(QIODevice::ReadOnly)) {
        sendHttpResponse(socket, 404, QByteArrayLiteral("Not Found"),
                         QByteArrayLiteral("text/plain; charset=utf-8"),
                         QByteArrayLiteral("Not Found\n"));
        return;
    }

    sendHttpResponse(socket, 200, QByteArrayLiteral("OK"),
                     contentTypeFor(path), resource.readAll());
}

void WebChannelServer::sendHttpResponse(QTcpSocket *socket, int status,
                                         const QByteArray &reason,
                                         const QByteArray &contentType,
                                         const QByteArray &body)
{
    if (!socket)
        return;
    QByteArray response;
    response += "HTTP/1.1 " + QByteArray::number(status) + " " + reason + "\r\n";
    response += "Content-Type: " + contentType + "\r\n";
    response += "Content-Length: " + QByteArray::number(body.size()) + "\r\n";
    response += "Cache-Control: no-store\r\n";
    response += "Access-Control-Allow-Origin: *\r\n";
    response += "Connection: close\r\n\r\n";
    response += body;
    socket->write(response);
    socket->disconnectFromHost();
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

void WebChannelServer::removeHttpSocket()
{
    auto *socket = qobject_cast<QTcpSocket *>(sender());
    if (!socket)
        return;
    m_httpRequests.remove(socket);
    socket->deleteLater();
}
