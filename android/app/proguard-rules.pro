# kotlinx.serialization — generatsiya qilingan serializerlar refleksiya
# orqali topiladi, shuning uchun ular saqlanadi.
-keepattributes *Annotation*, InnerClasses
-dontnote kotlinx.serialization.**
-keepclassmembers class **$Companion {
    kotlinx.serialization.KSerializer serializer(...);
}
-keepclasseswithmembers class ** {
    public static ** INSTANCE;
    kotlinx.serialization.KSerializer serializer(...);
}
