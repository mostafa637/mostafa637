# AVD research — 2026-08-15

## Sources and findings

1. Android Developers, [Configure hardware acceleration for the Android Emulator](https://developer.android.com/studio/run/emulator-acceleration): VM acceleration requires a host/system-image architecture match. The documented table says x86_64 hosts use x86/x86_64 images and ARM64 hosts use arm64-v8a images; ARM images on Intel/AMD cannot use the documented VM acceleration. The same page documents `-gpu software` and `-gpu swiftshader` for software graphics, and marks `swiftshader_indirect` deprecated in newer emulator releases.

2. ReactiveCircus issue [Support for M1 Silicon — HVF error: HV_UNSUPPORTED](https://github.com/ReactiveCircus/android-emulator-runner/issues/350): an arm64-v8a AVD on GitHub macOS can fail with `HVF error: HV_UNSUPPORTED` and `qemu-system-aarch64-headless: failed to initialize HVF: Invalid argument`, matching this project's AVD log.

3. GitHub Docs, [Larger runners reference](https://docs.github.com/en/actions/reference/runners/larger-runners): macOS larger runners are available as Intel x64 (`macos-latest-large`) and Apple Silicon arm64 (`macos-latest-xlarge`); macOS larger runners do not support nested virtualization.

## Project implication

The current workflow used `runs-on: macos-latest` with an `arm64-v8a` image. The log showed `qemu-system-aarch64-headless` and `HV_UNSUPPORTED`. To preserve the requested macOS AVD smoke test on the likely x64 standard runner, the next workflow revision should detect `uname -m`, select `x86_64`/`google_apis;x86_64` on x64 and `arm64-v8a` on arm64, create a matching AVD, and download the corresponding APK artifact. Use the non-deprecated `-gpu swiftshader` where supported, with a fallback only if necessary.

## Runner label decision

تُظهر مراجع GitHub الرسمية أن `macos-latest` هو runner macOS arm64، وأن nested virtualization غير مدعوم على runners macOS arm64. كما يعلن مستودع `actions/runner-images` أن `macos-15-intel` و`macos-26-intel` هما labels Intel المتاحة، وأن `macos-15-intel` هو label الانتقال من macOS 13 حتى أغسطس 2027. لذلك لا يمكن تنفيذ AVD x86_64 software test فعليًا على `macos-latest` arm64، ولا يمكن تشغيل AVD arm64 بسبب HVF unsupported؛ سيُنقل smoke test الفعلي إلى `macos-15-intel` مع صورة x86_64 و`-no-accel`، مع إبقاء البناء Android arm64-v8a وx86_64 مستقلًا.

المراجع: [GitHub-hosted runners reference](https://docs.github.com/en/actions/reference/runners/github-hosted-runners)، [runner-images issue #13045](https://github.com/actions/runner-images/issues/13045)، [runner-images README](https://github.com/actions/runner-images).

## Android Emulator Runner

توضح صفحة [ReactiveCircus Android Emulator Runner](https://github.com/marketplace/actions/android-emulator-runner) أن الإجراء يثبت SDK وAVD ويشغّل المحاكي وينتظر اكتمال boot ثم ينفذ script المستخدم. وتوصي الصفحة باستخدام `ubuntu-latest`/runners Linux الأكبر لتسريع hardware-accelerated emulators، مع تفعيل صلاحيات `/dev/kvm` بقاعدة udev. كما توضح أن الصور الحديثة Intel `x86`/`x86_64` تعتمد على تسريع VM، وأن خيار `arch: x86_64` مناسب لصورة API 35؛ أما ARM-based emulators القديمة فليست المسار الموصى به. هذا يدعم job Linux+KVM المضاف إلى workflow.
