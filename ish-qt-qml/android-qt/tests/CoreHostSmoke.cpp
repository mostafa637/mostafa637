#include "../src/CoreSession.h"
#include "RootfsManager.h"

#include <QCoreApplication>
#include <QElapsedTimer>
#include <QThread>

#include <cstdio>

int main(int argc, char **argv)
{
    QCoreApplication application(argc, argv);
    QCoreApplication::setOrganizationName(QStringLiteral("iSH"));
    QCoreApplication::setApplicationName(QStringLiteral("ish-core-host-smoke"));

    RootfsManager rootfs;
    rootfs.prepare();
    if (!rootfs.prepared()) {
        std::fprintf(stderr, "Rootfs preparation failed\n");
        return 1;
    }

    CoreSession session;
    QByteArray output;
    int exitCode = 0;
    bool exited = false;
    QString error;

    QObject::connect(&session, &CoreSession::outputReady,
                     [&](const QByteArray &bytes) { output += bytes; });
    QObject::connect(&session, &CoreSession::exited,
                     [&](int code) {
                         exited = true;
                         exitCode = code;
                     });
    QObject::connect(&session, &CoreSession::errorOccurred,
                     [&](const QString &message) { error = message; });

    if (!session.start(rootfs.rootPath(), {QStringLiteral("/bin/sh")}, {})) {
        std::fprintf(stderr, "Unable to start CoreSession\n");
        return 2;
    }

    const QByteArray command =
        "if grep -q '^nameserver[[:space:]]' /etc/resolv.conf; then "
        "printf 'ISH_RESOLV_PRESENT=1\\n'; "
        "else printf 'ISH_RESOLV_PRESENT=0\\n'; fi; "
        "apk add --no-progress python3; rc=$?; "
        "printf '\\nISH_APK_RESULT=%s\\n' \"$rc\"; "
        "if command -v python3 >/dev/null 2>&1; then "
        "printf 'ISH_PYTHON_PRESENT=1\\n'; python3 --version; "
        "else printf 'ISH_PYTHON_PRESENT=0\\n'; fi; "
        "printf 'ISH_APK_'\"DONE\"'\\n'\n";
    if (session.write(command) != command.size()) {
        std::fprintf(stderr, "Unable to send apk test command to CoreSession\n");
        session.stop();
        return 3;
    }

    QElapsedTimer timer;
    timer.start();
    const qint64 configuredTimeout = qEnvironmentVariable("ISH_SMOKE_TIMEOUT_MS").toLongLong();
    const qint64 smokeTimeout = configuredTimeout > 0 ? configuredTimeout : 180000;
    while (!output.contains("ISH_APK_DONE") && !exited && timer.elapsed() < smokeTimeout) {
        application.processEvents(QEventLoop::AllEvents, 50);
        QThread::msleep(20);
    }
    application.processEvents(QEventLoop::AllEvents, 100);

    const bool passed = output.contains("ISH_RESOLV_PRESENT=1") &&
                        output.contains("ISH_APK_RESULT=0") &&
                        output.contains("ISH_PYTHON_PRESENT=1") &&
                        output.contains("Python 3.");
    session.stop();
    application.processEvents(QEventLoop::AllEvents, 100);

    if (!passed) {
        std::fprintf(stderr,
                     "Host CoreSession Qt smoke test failed: exit=%d elapsed=%lldms error=%s\n",
                     exitCode, timer.elapsed(), error.toUtf8().constData());
        std::fprintf(stderr, "Captured core output:\n%.*s\n",
                     static_cast<int>(output.size()), output.constData());
        return 4;
    }

    std::fprintf(stdout, "LOCAL_CORESESSION_QT_SMOKE=PASS\n");
    std::fprintf(stdout, "ISH_RESOLV_PRESENT=1\n");
    std::fprintf(stdout, "ISH_APK_RESULT=0\n");
    std::fprintf(stdout, "ISH_PYTHON_PRESENT=1\n");
    return 0;
}
