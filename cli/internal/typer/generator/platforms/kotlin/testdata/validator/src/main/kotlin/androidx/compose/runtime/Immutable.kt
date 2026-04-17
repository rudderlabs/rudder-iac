package androidx.compose.runtime

// Stub of androidx.compose.runtime.Immutable for compile-only validation of the
// composeImmutable generator option. The real annotation ships with Jetpack
// Compose; consumers must depend on androidx.compose.runtime themselves.
@Target(AnnotationTarget.CLASS)
@Retention(AnnotationRetention.BINARY)
annotation class Immutable
