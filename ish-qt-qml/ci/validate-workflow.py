#!/usr/bin/env python3
from pathlib import Path

workflow = Path('.github/workflows/build-qt.yml').read_text(encoding='utf-8')
required = {
    'QT_VERSION: 6.11.1': 'Qt 6.11.1',
    'aqtinstall': 'aqtinstall',
    'runs-on: macos-latest': 'macOS runner',
    'test-android-avd:': 'AVD job',
    'system-images;android-35;google_apis;arm64-v8a': 'API 35 arm64 image',
    '-accel off': 'software emulator mode',
}
missing = [label for needle, label in required.items() if needle not in workflow]
if missing:
    raise SystemExit('Missing workflow requirements: ' + ', '.join(missing))
if workflow.count('test-android-avd:') != 1:
    raise SystemExit('Expected exactly one AVD job')
if workflow.count('runs-on: macos-latest') < 1:
    raise SystemExit('Expected a macOS runner')
cmake = Path('android-qt/CMakeLists.txt').read_text(encoding='utf-8')
if 'QT_ANDROID_PACKAGE_NAME "com.mostafa637.ishqt"' not in cmake:
    raise SystemExit('Missing stable Android package name in CMake')
print('workflow validation: OK')
