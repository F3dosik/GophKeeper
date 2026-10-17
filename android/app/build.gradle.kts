plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
}

android {
    namespace = "io.github.f3dosik.shron"
    compileSdk {
        version = release(37)
    }

    defaultConfig {
        applicationId = "io.github.f3dosik.shron"
        minSdk = 24
        targetSdk = 37
        versionCode = 1
        versionName = "1.0"

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
        // Каждый инструментальный тест — в новом процессе с пустыми данными приложения:
        // Go-ядро одно на процесс и хранит настройки и сессию в filesDir.
        testInstrumentationRunnerArguments["clearPackageData"] = "true"

        // Go-ядро собирается только для этих ABI (make android-aar): без фильтра APK ставился бы
        // и на 32-битные устройства, где нет libgojni.so, и падал при запуске.
        ndk {
            abiFilters += listOf("arm64-v8a", "x86_64")
        }

        // Соединение без TLS — только для разработки: ./gradlew -Pshron.insecure=true ...
        val insecure = providers.gradleProperty("shron.insecure").orNull == "true"
        buildConfigField("boolean", "SHRON_INSECURE", insecure.toString())
    }

    // Release, подписанный отладочным ключом, — только для smoke-теста в CI
    // (-Pshron.debugSignedRelease=true): настоящие релизы так не собираются.
    val debugSignedRelease = providers.gradleProperty("shron.debugSignedRelease").orNull == "true"

    buildTypes {
        release {
            if (debugSignedRelease) signingConfig = signingConfigs.getByName("debug")
            // R8 для всего приложения. Ограничение packageScope из шаблона Studio ломало
            // запуск: классы Kotlin внутри области переименовывались и закрывались, а код
            // приложения вне её ссылался на них (IllegalAccessError в ShronApplication).
            // Классы gomobile защищены правилами в src/main/keepRules.
            optimization {
                enable = true
            }
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_11
        targetCompatibility = JavaVersion.VERSION_11
    }
    testOptions {
        execution = "ANDROIDX_TEST_ORCHESTRATOR"
    }
    buildFeatures {
        compose = true
        buildConfig = true
    }
}

dependencies {
    // Go-ядро клиента (gomobile): собирается `make android-aar` из корня репозитория.
    implementation(files("libs/shron.aar"))
    implementation(platform(libs.androidx.compose.bom))
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.compose.material3)
    implementation(libs.androidx.compose.material.icons.core)
    implementation(libs.androidx.compose.ui)
    implementation(libs.androidx.compose.ui.graphics)
    implementation(libs.androidx.compose.ui.tooling.preview)
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.lifecycle.runtime.ktx)
    implementation(libs.androidx.lifecycle.runtime.compose)
    implementation(libs.androidx.lifecycle.process)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    testImplementation(libs.junit)
    // В android.jar для JVM-тестов org.json — заглушки; настоящая реализация для ShronError.
    testImplementation(libs.org.json)
    androidTestImplementation(platform(libs.androidx.compose.bom))
    androidTestImplementation(libs.androidx.compose.ui.test.junit4)
    androidTestImplementation(libs.androidx.espresso.core)
    androidTestImplementation(libs.androidx.junit)
    androidTestImplementation(libs.androidx.test.runner)
    androidTestUtil(libs.androidx.test.orchestrator)
    debugImplementation(libs.androidx.compose.ui.test.manifest)
    debugImplementation(libs.androidx.compose.ui.tooling)
}