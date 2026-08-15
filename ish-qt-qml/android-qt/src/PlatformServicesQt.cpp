#include "PlatformServicesQt.h"

#include <QClipboard>
#include <QGuiApplication>

PlatformServicesQt::PlatformServicesQt(QObject *parent)
    : QObject(parent)
{
}

QString PlatformServicesQt::pasteText() const
{
    const QClipboard *clipboard = QGuiApplication::clipboard();
    if (!clipboard)
        return {};
    return clipboard->text(QClipboard::Clipboard);
}

void PlatformServicesQt::reportError(const QString &title, const QString &message)
{
    emit errorReported(title, message);
}
