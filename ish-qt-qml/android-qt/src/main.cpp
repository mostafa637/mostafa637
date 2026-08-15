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

    webChannel.start();
    QObject::connect(&ishSession, &IshSession::outputReady,
                     &webChannel, &WebChannelServer::sendOutput);
    QObject::connect(&webChannel, &WebChannelServer::inputReceived,
                     &ishSession, [&ishSession](const QString &value) {
                         ishSession.sendInput(value);
                     });

    // The rootfs is extracted into the application data directory, not used
    // directly from the read-only QRC resource. This also makes the first
    // QML load deterministic on Android and Linux.
    rootfsManager.prepare();

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
    return app.exec();
}
