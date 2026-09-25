plugins {
    kotlin("jvm") version "1.9.22"
    application
}

group = "com.example"
version = "1.0.0"

val sdkVersion = "[1.1.0,2.0.0)"
val junitVersion = "5.9.3"

repositories {
    mavenCentral()
}

dependencies {
    implementation("com.rudderstack.sdk.kotlin:core:$sdkVersion")

    // Test dependencies
    testImplementation(kotlin("test"))
    testImplementation("org.junit.jupiter:junit-jupiter:$junitVersion")
}

tasks.test {
    useJUnitPlatform()

    testLogging {
        events("passed", "skipped", "failed")
        exceptionFormat = org.gradle.api.tasks.testing.logging.TestExceptionFormat.FULL
        showExceptions = true
        showCauses = true
        showStackTraces = true
    }
}

kotlin {
    jvmToolchain(21)
}