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

You implement `RudderTyperTransport` once per platform, on top of that platform's SDK. Shared code only sees `RudderTyper`.

The transport receives plain Kotlin values (`String`, `Boolean`, `Long`, `Double`, `Map`, `List`, `null`), so the interface also works from Swift. `context` holds the `ruddertyper` context (platform, CLI version, tracking plan ID and version) and any `customContext` the caller passed. Send it as the event's custom context, which both SDKs accept in the options of an event call.

### Android, with the Kotlin SDK

Add `com.rudderstack.sdk.kotlin:android` to the Android module. The Kotlin SDK takes `JsonObject` values, so the adapter converts the maps first.

```kotlin
import android.app.Application
import com.example.analytics.RudderTyperTransport
import com.rudderstack.sdk.kotlin.android.Analytics
import com.rudderstack.sdk.kotlin.android.Configuration
import com.rudderstack.sdk.kotlin.core.internals.logger.Logger
import com.rudderstack.sdk.kotlin.core.internals.models.RudderOption
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive

/** Implements the generated transport on top of the RudderStack Kotlin SDK. */
class AndroidTransport(
    application: Application,
    writeKey: String,
    dataPlaneUrl: String,
    controlPlaneUrl: String = "https://api.rudderstack.com",
) : RudderTyperTransport {
    private val analytics = Analytics(
        Configuration(
            writeKey = writeKey,
            application = application,
            dataPlaneUrl = dataPlaneUrl,
            controlPlaneUrl = controlPlaneUrl,
            logLevel = Logger.LogLevel.DEBUG,
        )
    )

    override fun track(name: String, properties: Map<String, Any?>, context: Map<String, Any?>) {
        analytics.track(name, properties.toJsonObject(), context.toOption())
    }

    override fun identify(userId: String, traits: Map<String, Any?>, context: Map<String, Any?>) {
        analytics.identify(userId, traits.toJsonObject(), context.toOption())
    }

    override fun group(groupId: String, traits: Map<String, Any?>, context: Map<String, Any?>) {
        analytics.group(groupId, traits.toJsonObject(), context.toOption())
    }

    override fun screen(screenName: String, category: String, properties: Map<String, Any?>, context: Map<String, Any?>) {
        analytics.screen(screenName, category, properties.toJsonObject(), context.toOption())
    }
}

private fun Map<String, Any?>.toOption() = RudderOption(customContext = toJsonObject())

private fun Map<String, Any?>.toJsonObject() = JsonObject(mapValues { it.value.toJsonElement() })

private fun Any?.toJsonElement(): JsonElement = when (this) {
    null -> JsonNull
    is JsonElement -> this
    is Boolean -> JsonPrimitive(this)
    is Number -> JsonPrimitive(this)
    is String -> JsonPrimitive(this)
    is Map<*, *> -> JsonObject(entries.associate { (k, v) -> k.toString() to v.toJsonElement() })
    is Iterable<*> -> JsonArray(map { it.toJsonElement() })
    else -> JsonPrimitive(toString())
}
```

### iOS, with the Swift SDK

Add the `rudder-sdk-swift` package to the iOS app, and link the shared Kotlin module as a framework named `Shared`. Kotlin exports `RudderTyperTransport` as a Swift protocol, and the `Map` values arrive as `[String: Any]`.

```swift
import Foundation
import RudderStackAnalytics
import Shared

/// Implements the generated Kotlin RudderTyperTransport protocol on top of the RudderStack Swift SDK.
final class IosTransport: RudderTyperTransport {
    private let analytics: Analytics

    init(writeKey: String, dataPlaneUrl: String, controlPlaneUrl: String) {
        analytics = Analytics(configuration: Configuration(
            writeKey: writeKey,
            dataPlaneUrl: dataPlaneUrl,
            controlPlaneUrl: controlPlaneUrl,
            logLevel: .debug
        ))
    }

    func track(name: String, properties: [String: Any], context: [String: Any]) {
        analytics.track(name: name, properties: properties, options: RudderOption(customContext: context))
    }

    func identify(userId: String, traits: [String: Any], context: [String: Any]) {
        analytics.identify(userId: userId, traits: traits, options: RudderOption(customContext: context))
    }

    func group(groupId: String, traits: [String: Any], context: [String: Any]) {
        analytics.group(groupId: groupId, traits: traits, options: RudderOption(customContext: context))
    }

    func screen(screenName: String, category: String, properties: [String: Any], context: [String: Any]) {
        analytics.screen(screenName: screenName, category: category, properties: properties, options: RudderOption(customContext: context))
    }
}
```

Create the transport in each app and hand it to the shared code, for example through a Compose Multiplatform entry point that takes a `RudderTyperTransport`.

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

The adapters above were built and run with Kotlin 2.2.21, Compose Multiplatform 1.9.3, Android Gradle Plugin 8.12.3, Kotlin SDK 2.2.0 and Swift SDK 2.1.0, on an Android emulator and an iPhone simulator. They have not been run on a physical iOS device.

To generate without credentials on the build machine, use a snapshot file. See [GRADLE.md](GRADLE.md).
