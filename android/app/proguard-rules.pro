# Referenced by the `release` build type in app/build.gradle
# (proguardFiles getDefaultProguardFile('proguard-android-optimize.txt'), 'proguard-rules.pro').
# The file has to exist or `assembleRelease` fails on a missing-file error, even
# though minifyEnabled is false today.

# term.js calls these through webView.addJavascriptInterface(this, "Android");
# R8 must not rename or strip them, or hterm's native.load() silently stops
# reaching the app and the terminal never reports itself loaded.
-keepclassmembers class com.ish.emulator.MainActivity {
    @android.webkit.JavascriptInterface <methods>;
}
-keepattributes *Annotation*
