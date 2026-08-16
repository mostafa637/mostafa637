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
#include <QGuiApplication>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QUrl>

int main(int argc, char *argv[])
{
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
