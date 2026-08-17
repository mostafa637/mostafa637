#include "IshSession.h"

#include <QDebug>

IshSession::IshSession(QObject *parent)
    : QObject(parent),
      m_core(new CoreSession(this))
{
    m_currentStyle.insert(QStringLiteral("fontFamily"), QStringLiteral("Noto Sans Mono"));
    m_currentStyle.insert(QStringLiteral("fontSize"), 14);
    m_currentStyle.insert(QStringLiteral("foregroundColor"), QStringLiteral("#f5f5f7"));
    m_currentStyle.insert(QStringLiteral("backgroundColor"), QStringLiteral("#000000"));
    m_currentStyle.insert(QStringLiteral("selectionColor"), QStringLiteral("#3a78d4"));
    m_currentStyle.insert(QStringLiteral("selectedTextColor"), QStringLiteral("#ffffff"));
    m_currentStyle.insert(QStringLiteral("blinkCursor"), true);
    m_currentStyle.insert(QStringLiteral("cursorShape"), QStringLiteral("block"));

    connect(m_core, &CoreSession::outputReady,
            this, &IshSession::handleOutput);
    connect(m_core, &CoreSession::exited,
            this, &IshSession::handleState);
    connect(m_core, &CoreSession::errorOccurred,
            this, &IshSession::handleCoreError);
}

IshSession::~IshSession()
{
    stop();
}

void IshSession::configure(const QString &rootPath,
                           const QStringList &bootCommand,
                           const QStringList &launchCommand)
{
    if (m_alive)
        stop();
    m_rootPath = rootPath;
    m_bootCommand = bootCommand;
    m_launchCommand = launchCommand;
}

bool IshSession::start(const QString &rootPath,
                       const QStringList &bootCommand,
                       const QStringList &launchCommand)
{
    configure(rootPath, bootCommand, launchCommand);
    m_stopRequested = false;
    if (!m_core->start(m_rootPath, m_bootCommand, m_launchCommand))
        return false;
    setAlive(true);
    qWarning() << "[ish-qt] IshSession::start SUCCEEDED alive=" << m_alive;
    return true;
}

void IshSession::stop()
{
    m_stopRequested = true;
    if (m_core != nullptr)
        m_core->stop();
    setAlive(false);
}

void IshSession::load()
{
    emit styleChanged(m_currentStyle);
    emit loaded();
}

void IshSession::sendInput(const QString &value)
{
    if (!m_core || !m_alive || value.isEmpty())
        return;
    m_core->write(value.toUtf8());
}

void IshSession::resize(int columns, int rows)
{
    if (m_core != nullptr && m_alive)
        m_core->resize(columns, rows);
}

void IshSession::controlModifierConsumed()
{
    emit controlModifierConsumedSignal();
}

void IshSession::setStyle(const QVariantMap &style)
{
    if (style == m_currentStyle)
        return;
    m_currentStyle = style;
    emit styleChanged(m_currentStyle);
}

void IshSession::handleOutput(const QByteArray &bytes)
{
    if (!bytes.isEmpty())
        emit outputReady(QString::fromUtf8(bytes));
}

void IshSession::handleState(int exitCode)
{
    qWarning() << "[ish-qt] IshSession::handleState exitCode=" << exitCode << "stopRequested=" << m_stopRequested;
    setAlive(false);
    if (!m_stopRequested && exitCode != 0)
        emit sessionError(QStringLiteral("iSH session ended with status %1").arg(exitCode));
}

void IshSession::handleCoreError(const QString &message)
{
    emit sessionError(message);
}

void IshSession::setAlive(bool value)
{
    if (m_alive == value)
        return;
    m_alive = value;
    qWarning() << "[ish-qt] IshSession aliveChanged:" << m_alive;
    emit aliveChanged(m_alive);
}
