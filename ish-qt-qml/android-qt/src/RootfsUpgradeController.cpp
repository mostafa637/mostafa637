#include "RootfsUpgradeController.h"
#include "RootfsManager.h"

RootfsUpgradeController::RootfsUpgradeController(RootfsManager *manager, QObject *parent)
    : QObject(parent),
      m_manager(manager)
{
    if (!m_manager)
        return;
    connect(m_manager, &RootfsManager::progressChanged,
            this, &RootfsUpgradeController::onManagerProgress);
    connect(m_manager, &RootfsManager::preparationError,
            this, &RootfsUpgradeController::onManagerError);
    connect(m_manager, &RootfsManager::preparedChanged,
            this, &RootfsUpgradeController::finishIfPrepared);
}

void RootfsUpgradeController::reinstallBundledRootfs()
{
    if (m_busy || !m_manager)
        return;
    setBusy(true);
    setProgress(0);
    setMessage(QStringLiteral("Removing installed rootfs…"));
    m_manager->resetInstalledData();
    setMessage(QStringLiteral("Installing bundled rootfs…"));
    m_manager->prepare();
    if (m_manager->prepared())
        finishIfPrepared();
}

void RootfsUpgradeController::onManagerProgress(int percent, const QString &message)
{
    if (!m_busy)
        return;
    setProgress(percent);
    setMessage(message);
}

void RootfsUpgradeController::onManagerError(const QString &message)
{
    if (!m_busy)
        return;
    setMessage(message);
    setBusy(false);
    emit failed(message);
}

void RootfsUpgradeController::finishIfPrepared()
{
    if (!m_busy || !m_manager || !m_manager->prepared())
        return;
    setProgress(100);
    setMessage(QStringLiteral("Rootfs is ready"));
    setBusy(false);
}

void RootfsUpgradeController::setBusy(bool value)
{
    if (m_busy == value)
        return;
    m_busy = value;
    emit busyChanged();
}

void RootfsUpgradeController::setProgress(int value)
{
    const int bounded = qBound(0, value, 100);
    if (m_progress == bounded)
        return;
    m_progress = bounded;
    emit progressChanged();
}

void RootfsUpgradeController::setMessage(const QString &value)
{
    if (m_message == value)
        return;
    m_message = value;
    emit messageChanged();
}
