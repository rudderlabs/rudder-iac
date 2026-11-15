# RudderTyper on Compose Multiplatform

An Android app and an iOS app that share one Compose screen. The screen sends typed events through `RudderTyper`, generated with `emit=common`, so shared code never imports an SDK class. Each platform provides a small `RudderTyperTransport` on top of its own SDK. See [typer/docs/KOTLIN_MULTIPLATFORM.md](../../docs/KOTLIN_MULTIPLATFORM.md).

```
shared/src/commonMain   Compose screen and the generated client (typer/RudderTyper.kt)
shared/src/androidMain  AndroidTransport, on the RudderStack Kotlin SDK
shared/src/iosMain      MainViewController, which receives the Swift transport
androidApp              Android entry point
iosApp                  SwiftUI entry point and IosTransport, on the RudderStack Swift SDK
```

## Regenerate the client

The checked-in client came from a small local spec with two events. To regenerate it from your own plan:

```bash
rudder-cli typer generate --platform kotlin --option emit=common \
  --option packageName=com.example.typerapp.typer --option outputFileName=RudderTyper.kt \
  --tracking-plan-id <id> -o shared/src/commonMain/kotlin/com/example/typerapp/typer
```

Then update the event calls in `App.kt` to the generated method and parameter names.

## Run on Android

Needs JDK 17 or newer and the Android SDK. Point `local.properties` at the SDK (`sdk.dir=...`).

```bash
./gradlew :androidApp:assembleDebug
```

`MainActivity` sends to `http://10.0.2.2:19187`, the host machine as seen from the emulator. Run any HTTP server there that answers `GET /sourceConfig` with a source object and accepts `POST /v1/batch`, or change the URLs to a real data plane and write key.

## Run on iOS

Needs Xcode and [XcodeGen](https://github.com/yonaskolb/XcodeGen).

```bash
cd iosApp && xcodegen generate && open iosApp.xcodeproj
```

The iOS app builds with Xcode 27 and runs on the iPhone 17 simulator, and its events reach a data plane with the expected plan stamp. Xcode does not inherit your shell `JAVA_HOME`, and the Android Gradle plugin needs JDK 17 or later, so build with `RUDDER_JAVA_HOME=<jdk17+ path> xcodebuild ...` or set it in the scheme's environment. The Info.plist sets `CADisableMinimumFrameDurationOnPhone`, which Compose requires on iPhone.
