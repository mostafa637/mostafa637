#include "FontCatalog.h"
#include "IshSession.h"
#include "PlatformServicesQt.h"
#include "RootFilesModel.h"
#include "RootModel.h"
#include "RootfsManager.h"
#include "RootfsUpgradeController.h"
#include "ThemeManager.h"
#include "UserPreferencesQt.h"
#include "WebChannelServer.h"

#include <QCoreApplication>
#include <QFontDatabase>
#include <csignal>
#include <QGuiApplication>
#include <QProcessEnvironment>
#include <QQuickWindow>
#include <QSGRendererInterface>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QUrl>
#include <QtWebView/QtWebView>

namespace {

// Qt WebView embeds Chromium, which dumps repetitive crashpad/gpu diagnostic
// lines into the process stderr. Capture this stderr early via the iSH core
// transport, so suppress the messages at the Chromium source when Chromium
// flags are accepted. Keep any user-provided flags intact.
void suppressChromiumDiagnostics()
{
    QProcessEnvironment env = QProcessEnvironment::systemEnvironment();
    const QString existing = env.value(QStringLiteral("QT_CHROMIUM_FLAGS")).simplified();
    QString flags = existing;
    const QStringList extra = {
        QStringLiteral("--no-sandbox"),
        QStringLiteral("--disable-gpu"),
        QStringLiteral("--disable-extensions")
    };
    for (const QString &flag : extra) {
        if (!existing.contains(flag))
            flags += ' ' + flag;
    }
    if (!flags.simplified().isEmpty())
        qputenv("QT_CHROMIUM_FLAGS", flags.simplified().toUtf8());
}

} // namespace

int main(int argc, char *argv[])
{
    // Qt WebView must be initialized before QGuiApplication, especially on Android.
    QtWebView::initialize();
    suppressChromiumDiagnostics();
#ifdef Q_OS_ANDROID
    // The Android emulator/older GPU drivers can corrupt clipped Qt Quick
    // batches into large gray triangles. The terminal UI is lightweight, and
    // the WebView has its own renderer, so use Qt Quick's software backend for
    // deterministic keyboard/tool-bar rendering on Android.
    QQuickWindow::setGraphicsApi(QSGRendererInterface::Software);
#endif
    // The iSH kernel runs as a pthread inside the Qt process. With the
    // default SIGINT disposition, Ctrl+C (or an inherited SIGINT) would be
    // delivered to the kernel thread, whose signal path dereferences an
    // uninitialised task->signal mutex and segfaults. Ignore SIGINT so the
    // kernel can only be stopped gracefully through the input pipe.
    std::signal(SIGINT, SIG_IGN);
    QGuiApplication app(argc, argv);
    QCoreApplication::setOrganizationName(QStringLiteral("iSH"));
    QCoreApplication::setOrganizationDomain(QStringLiteral("ish.app"));
    QCoreApplication::setApplicationName(QStringLiteral("iSH Qt"));
    QCoreApplication::setApplicationVersion(QStringLiteral("1.0"));

    const int bundledFont = QFontDatabase::addApplicationFont(
        QStringLiteral(":/ish-assets/fonts/NotoSansMono-Regular.ttf"));
    Q_UNUSED(bundledFont);

    ThemeManager themes;
    UserPreferencesQt preferences(&themes);
    RootfsManager rootfsManager;
    WebChannelServer webChannel;
    PlatformServicesQt platformServices;
    FontCatalog fontCatalog;
    RootModel rootModel;
    RootFilesModel rootFilesModel;
    RootfsUpgradeController rootfsUpgrade(&rootfsManager);
    IshSession ishSession;

    platformServices.installCrashHandler();
    platformServices.logDiagnostic(QStringLiteral("startup"),
                                   QStringLiteral("Qt application initialized; log path: %1")
                                       .arg(platformServices.diagnosticLogPath()));
    QObject::connect(&ishSession, &IshSession::sessionError,
                     &platformServices, [&platformServices](const QString &message) {
                         platformServices.logDiagnostic(QStringLiteral("native session"), message);
                     });
    QObject::connect(&webChannel, &WebChannelServer::serverError,
                     &platformServices, [&platformServices](const QString &message) {
                         platformServices.logDiagnostic(QStringLiteral("web channel"), message);
                     });
    QObject::connect(&rootfsManager, &RootfsManager::preparationError,
                     &platformServices, [&platformServices](const QString &message) {
                         platformServices.logDiagnostic(QStringLiteral("rootfs preparation"), message);
                     });

    webChannel.start();
    QObject::connect(&ishSession, &IshSession::outputReady,
                     &webChannel, &WebChannelServer::sendOutput);
    QObject::connect(&webChannel, &WebChannelServer::inputReceived,
                     &ishSession, [&ishSession](const QString &value) {
                         ishSession.sendInput(value);
                     });

    QQmlApplicationEngine engine;
    QQmlContext *context = engine.rootContext();
    context->setContextProperty(QStringLiteral("ishSession"), &ishSession);
    context->setContextProperty(QStringLiteral("rootfsManager"), &rootfsManager);
    context->setContextProperty(QStringLiteral("preferences"), &preferences);
    context->setContextProperty(QStringLiteral("themes"), &themes);
    context->setContextProperty(QStringLiteral("webChannel"), &webChannel);
    context->setContextProperty(QStringLiteral("platformServices"), &platformServices);
    context->setContextProperty(QStringLiteral("fontCatalog"), &fontCatalog);
    context->setContextProperty(QStringLiteral("rootModel"), &rootModel);
    context->setContextProperty(QStringLiteral("rootFilesModel"), &rootFilesModel);
    context->setContextProperty(QStringLiteral("rootfsUpgrade"), &rootfsUpgrade);

    engine.loadFromModule(QStringLiteral("IshQt"), QStringLiteral("Main"));
    if (engine.rootObjects().isEmpty())
        return 1;

    // Load QML before preparing rootfs so that progress and preparationError
    // signals are visible in the UI. The core still starts only after the
    // manager emits preparedChanged().
    rootfsManager.prepare();
    return app.exec();
}
