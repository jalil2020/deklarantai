package uz.deklarant.ai

import android.app.Application
import uz.deklarant.ai.di.AppContainer

/**
 * Ilova darajasidagi yagona bog'liqliklar egasi.
 *
 * Konteyner shu yerda yashaydi, chunki HTTP klient ilova umri davomida
 * bitta bo'lishi kerak — Activity qayta yaratilganda (ekran burilishi)
 * u qayta yasalmasin.
 */
class DeklarantApp : Application() {

    lateinit var container: AppContainer
        private set

    override fun onCreate() {
        super.onCreate()
        container = AppContainer(this)
    }

    override fun onTerminate() {
        container.close()
        super.onTerminate()
    }
}
