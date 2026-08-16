#include "PlatformServicesQt.h"

#include <QClipboard>
#include <QGuiApplication>

PlatformServicesQt::PlatformServicesQt(QObject *parent)
    : QObject(parent)
{
    if (QGuiApplication::clipboard()) {
        connect(QGuiApplication::clipboard(), &QClipboard::dataChanged,
                this, &PlatformServicesQt::clipboardChanged);
    }
}

QString PlatformServicesQt::pasteText() const
{
    return clipboardText();
}

QString PlatformServicesQt::clipboardText() const
{
    const QClipboard *clipboard = QGuiApplication::clipboard();
    if (!clipboard)
        return {};
    return clipboard->text(QClipboard::Clipboard);
}

void PlatformServicesQt::copyText(const QString &text)
{
    QClipboard *clipboard = QGuiApplication::clipboard();
    if (!clipboard)
        return;
    clipboard->setText(text, QClipboard::Clipboard);
}

void PlatformServicesQt::clearClipboard()
{
    QClipboard *clipboard = QGuiApplication::clipboard();
    if (!clipboard)
        return;
    clipboard->clear(QClipboard::Clipboard);
}

bool PlatformServicesQt::hasClipboardText() const
{
    return !clipboardText().isEmpty();
}

void PlatformServicesQt::reportError(const QString &title, const QString &message)
{
    emit errorReported(title, message);
}
