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
- `RudderTyperOptions`, the per-event custom context and integrations that every transport method receives.

## What you write

You implement `RudderTyperTransport` once per platform, on top of that platform's SDK. Shared code only sees `RudderTyper`.

Every transport method ends with a `RudderTyperOptions` parameter, a class with two fields. `context` is custom context for the event, and it already holds the `ruddertyper` context (platform, CLI version, tracking plan ID and version) next to any custom context the caller passed. `integrations` turns destinations on or off for the event. Attach both to the SDK's event options. Because the options are a class, a later release can add a field without breaking the transports you wrote.

The maps hold plain Kotlin values (`String`, `Boolean`, `Long`, `Double`, `Map`, `List`, `null`).

### Android, with the Kotlin SDK

Add `com.rudderstack.sdk.kotlin:android` to the Android module. The Kotlin SDK takes `JsonObject` values, so the adapter converts the maps first.

```kotlin
import android.app.Application
import com.example.analytics.RudderTyperOptions
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

    override fun track(name: String, properties: Map<String, Any?>, options: RudderTyperOptions) {
        analytics.track(name, properties.toJsonObject(), options.toOption())
    }

    override fun identify(userId: String, traits: Map<String, Any?>, options: RudderTyperOptions) {
        analytics.identify(userId, traits.toJsonObject(), options.toOption())
    }

    override fun group(groupId: String, traits: Map<String, Any?>, options: RudderTyperOptions) {
        analytics.group(groupId, traits.toJsonObject(), options.toOption())
    }

    override fun screen(screenName: String, category: String, properties: Map<String, Any?>, options: RudderTyperOptions) {
        analytics.screen(screenName, category, properties.toJsonObject(), options.toOption())
    }
}

private fun RudderTyperOptions.toOption() =
    RudderOption(integrations = integrations.toJsonObject(), customContext = context.toJsonObject())

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

Add the `rudder-sdk-swift` package to the iOS app, and link the shared Kotlin module as a framework named `Shared`. Kotlin exports `RudderTyperTransport` as a Swift protocol, and the maps arrive as `[String: Any]`.

Convert every Kotlin `Boolean` to a Swift `Bool` before you pass the values to the SDK. Swift receives a Kotlin `Boolean` as `KotlinBoolean`, a subclass of `NSNumber`, and the Swift SDK encodes an `NSNumber` as an integer, so `true` would go out as `1`. The `plainMap` function below does the conversion, including for nested maps and lists.

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

    func track(name: String, properties: [String: Any], options: RudderTyperOptions) {
        analytics.track(name: name, properties: plainMap(properties), options: options.toOption())
    }

    func identify(userId: String, traits: [String: Any], options: RudderTyperOptions) {
        analytics.identify(userId: userId, traits: plainMap(traits), options: options.toOption())
    }

    func group(groupId: String, traits: [String: Any], options: RudderTyperOptions) {
        analytics.group(groupId: groupId, traits: plainMap(traits), options: options.toOption())
    }

    func screen(screenName: String, category: String, properties: [String: Any], options: RudderTyperOptions) {
        analytics.screen(screenName: screenName, category: category, properties: plainMap(properties), options: options.toOption())
    }
}

private extension RudderTyperOptions {
    func toOption() -> RudderOption {
        RudderOption(integrations: plainMap(integrations), customContext: plainMap(context))
    }
}

/// Kotlin Boolean values reach Swift as KotlinBoolean, an NSNumber. The Swift SDK encodes an NSNumber
/// as an integer, so a Kotlin `true` would go out as 1. Unwrap them to Swift Bool, recursively.
private func plainMap(_ map: [String: Any]) -> [String: Any] {
    map.mapValues(plainValue)
}

private func plainValue(_ value: Any) -> Any {
    if let boolean = value as? KotlinBoolean { return boolean.boolValue }
    if let map = value as? [String: Any] { return plainMap(map) }
    if let list = value as? [Any] { return list.map(plainValue) }
    return value
}
```

Create the transport in each app and pass it to your shared code, for example through the entry point of a Compose Multiplatform screen.

## Switching between workspaces

Tracking plan IDs and versions differ per workspace, while the event models are the same. `RudderTyper` takes the plan ID and version as constructor arguments, with the generated values as defaults:

```kotlin
val typer = RudderTyper(transport, trackingPlanId = "tp_staging_plan", trackingPlanVersion = 9)
```

Pick the pair at runtime, the same way you pick the data plane URL. The models only match if both workspaces have the same event shapes at the versions you choose.

The version has to exist for that plan. RudderStack validates an event against the plan version it names, and drops an event that names a version the plan doesn't have, without reporting a violation.

## Differences from `emit=all`

- `RudderTyper` methods take `options: RudderTyperOptions` where `RudderAnalytics` methods take the SDK's `RudderOption`. It carries custom context and integrations, which are the parts of `RudderOption` that the typed client uses.
- `identify` and `group` always send traits through the `traits` argument of the transport, whichever identity section the rule uses.
- The generated file imports `kotlin.jvm.JvmName`, because common code has no implicit `kotlin.jvm` imports.

## Tested with

The adapters above were built and run with Kotlin 2.2.21, Compose Multiplatform 1.9.3, Android Gradle Plugin 8.12.3, Kotlin SDK 2.2.0 and Swift SDK 2.1.0, on an Android emulator and an iPhone simulator. They have not been run on a physical iOS device. A typed event with a boolean property, custom context and integrations was sent from the iPhone simulator, and the data plane received `true`, not `1`.

To generate without credentials on the build machine, use a snapshot file. See [GRADLE.md](GRADLE.md).
