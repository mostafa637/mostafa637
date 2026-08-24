# بناء Android عبر GitHub Actions

يحتوي المستودع على workflow باسم **Android** في `.github/workflows/android.yml`. يعمل workflow عند الدفع إلى أي فرع، وعند إنشاء Pull Request، ويمكن تشغيله يدوياً من تبويب Actions.

## ما يثبته البناء

| المكوّن | الإصدار أو المصدر | الغرض |
|---|---|---|
| Ubuntu | `ubuntu-24.04` | بيئة بناء موحدة على GitHub-hosted runner |
| Go | الإصدار المحدد في `go-pure/go.mod` | بناء واختبار التطبيق |
| JDK | Temurin 17 | متطلبات أدوات Android وGio |
| Android platform | `android-35` | واجهات Android المستخدمة أثناء التغليف |
| Android build-tools | `35.0.0` | إنشاء APK |
| Android NDK | `26.3.11579264` | متطلبات التغليف الأصلية التي قد تستعملها Gio |
| gogio | `gioui.org/cmd/gogio@v0.8.0` | تحويل تطبيق Gio إلى APK |

يقسم التنفيذ إلى وظيفتين. وظيفة `verify` تشغل فحص التنسيق واختبارات الحزم الأساسية مع `CGO_ENABLED=0`. كما تشغل فاحص modularity وتعرض مخالفاته كتحذير غير حاجب، لأن بعض ملفات JIT القديمة ما زالت تتجاوز حد الدالة/الملف وتحتاج دفعة refactoring مستقلة. بعد نجاح الاختبارات الأساسية تبدأ وظيفة `build`، فتجهز Android SDK وNDK ثم تنفذ `make android` وترفع `go-pure/bin/ishgo.apk` كـ artifact باسم `ishgo-android-apk`.

## تنزيل APK

افتح تبويب **Actions** في مستودع GitHub، اختر تشغيل **Android** المطلوب، ثم افتح التشغيل الناجح ونزّل artifact باسم `ishgo-android-apk`. هذا artifact مخصص للاختبار وليس توقيع إصدار متجر؛ نشره في Google Play يتطلب لاحقاً إعداد مفتاح توقيع آمن وملفات أسرار خارج المستودع.

## ملاحظات Pure Go

يظل التحقق الأساسي منفصلاً عن عملية التغليف. فشل بناء Android لا يلغي نتيجة اختبارات `test-core`، كما أن اجتياز `test-core` يثبت فقط أن الحزم الأساسية تعمل دون CGo في بيئة Linux/amd64؛ ولا يثبت وحده توافق كل واجهات Android أو كل معمارية هاتف.

إذا احتاج المشروع لاحقاً إلى بنية release موقعة، ينبغي إضافة job مستقل يعتمد على `build` ويقرأ keystore من GitHub Secrets، مع عدم تخزين keystore أو كلمات المرور داخل المستودع.

## تشغيل محلي مطابق قدر الإمكان

بعد تثبيت Android SDK وNDK و`gogio` محلياً، يمكن استخدام الأهداف الموجودة في Makefile:

```sh
cd go-pure
make test-core
make check-android
make android
```

يتطلب الهدف `android` وجود `gogio` و`ANDROID_HOME` أو `ANDROID_SDK_ROOT`. أما GitHub Actions فيثبت هذه المتطلبات تلقائياً داخل runner جديد في كل تشغيل.
