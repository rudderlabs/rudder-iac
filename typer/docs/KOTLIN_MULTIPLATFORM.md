# Kotlin Multiplatform and pure JVM modules

The default Kotlin output (`emit=all`) has a `RudderAnalytics` client that calls the RudderStack Kotlin SDK. That SDK targets Android and the JVM, so the file does not compile in Kotlin Multiplatform common code or for iOS.

`--option emit=common` writes a file with no SDK dependency:

```bash
rudder-cli typer generate --platform kotlin --option emit=common \
  --option packageName=com.example.analytics --snapshot ./tracking-plan.snapshot.json -o shared/src/commonMain/kotlin/com/example/analytics
```

The file needs only `kotlinx-serialization-json`. It contains:

- The models: data classes, enums, sealed classes and `rudderSerialize()` functions.
- `RudderTyper`, a client with one typed method per event, such as `trackProductClicked(properties)`.
- `RudderTyperTransport`, a four-method interface (`track`, `identify`, `group`, `screen`) that `RudderTyper` calls.

## What you write

You implement `RudderTyperTransport` once per platform, on top of that platform's SDK. In the example project the Android adapter is 62 lines, most of them a helper that converts maps to `JsonObject`, and the Swift adapter is 33 lines. Shared code only sees `RudderTyper`.

The transport receives plain Kotlin values (`String`, `Boolean`, `Long`, `Double`, `Map`, `List`, `null`), so the interface also works from Swift. `context` holds the `ruddertyper` context (platform, CLI version, tracking plan ID and version) and any `customContext` the caller passed. Send it as the event's custom context.

Android, with the Kotlin SDK:

```kotlin
class AndroidTransport(private val analytics: Analytics) : RudderTyperTransport {
    override fun track(name: String, properties: Map<String, Any?>, context: Map<String, Any?>) {
        analytics.track(name, properties.toJsonObject(), RudderOption(customContext = context.toJsonObject()))
    }
    // identify, group and screen follow the same pattern.
}
```

iOS, with the Swift SDK, implementing the Kotlin interface as a Swift protocol:

```swift
final class IosTransport: RudderTyperTransport {
    func track(name: String, properties: [String: Any], context: [String: Any]) {
        analytics.track(name: name, properties: properties, options: RudderOption(customContext: context))
    }
    // identify, group and screen follow the same pattern.
}
```

[`examples/compose-multiplatform`](../examples/compose-multiplatform) has both adapters and a Compose screen that calls the typed client from `commonMain`.

## Switching between workspaces

Tracking plan IDs and versions differ per workspace, while the event models are the same. `RudderTyper` takes the plan ID and version as constructor arguments, with the generated values as defaults:

```kotlin
val typer = RudderTyper(transport, trackingPlanId = "tp_staging_plan", trackingPlanVersion = 9)
```

Pick the pair at runtime, the same way you pick the data plane URL. The models only match if both workspaces have the same event shapes at the versions you choose.

## Differences from `emit=all`

- `RudderTyper` takes `customContext: Map<String, Any?>` where `RudderAnalytics` takes the SDK's `RudderOption`. Integration toggles are not available through it. Call the SDK directly for those.
- The generated file imports `kotlin.jvm.JvmName`, because common code has no implicit `kotlin.jvm` imports.

## Tested with

The example project builds and runs with Kotlin 2.2.21, Compose Multiplatform 1.9.3, Android Gradle Plugin 8.12.3, Kotlin SDK 2.2.0 (`com.rudderstack.sdk.kotlin:android`) and Swift SDK 2.1.0, on an Android emulator and an iPhone simulator. It has not been run on a physical iOS device. The Swift SDK receives the Kotlin `Map` values as `[String: Any]`.

To generate without credentials on the build machine, use a snapshot file. See [GRADLE.md](GRADLE.md).
