# Android WebView and icon fix notes

## External Qt references

Qt documents `QT_ANDROID_PACKAGE_SOURCE_DIR` as the path to a custom Android package template. The template may contain `AndroidManifest.xml`, Gradle files, and resources; androiddeployqt copies the default Qt template first and then copies the custom template over it. Source: [Qt 6.11 QT_ANDROID_PACKAGE_SOURCE_DIR](https://doc.qt.io/qt-6/cmake-target-property-qt-android-package-source-dir.html).

Qt's Android manifest documentation states that custom manifests must preserve Qt's metadata and insertion markers so androiddeployqt can add module permissions/features. It also documents `android.app.lib_name`, the Qt activity/application classes, and `-- %%INSERT_*%% --` placeholders. Source: [Qt 6.11 Android Manifest File Configuration](https://doc.qt.io/qt-6/android-manifest-file-configuration.html).

## Project findings

`WebChannelServer.cpp` generated `http://127.0.0.1:<port>/terminal/term.html`, which Android WebView rejected with `net::ERR_CLEARTEXT_NOT_PERMITTED`. The fix adds `android-qt/android/AndroidManifest.xml`, sets `android:usesCleartextTraffic="true"`, references `@xml/network_security_config`, and adds a permissive local network-security config. CMake sets `QT_ANDROID_PACKAGE_SOURCE_DIR` to the custom Android template.

The SVG files for all four arrow directions and both light/dark variants existed under `android-qt/assets/ui/icons`, but `assets.qrc` only packaged six of the sixteen arrow variants. QML references all four directions dynamically. The fix must add all missing arrow aliases to `assets.qrc` and keep the existing CI asset expectations.
