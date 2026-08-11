# cambios

1.Domain

- en chat.go quitar el IntentType(s) todos los que tengan que ver con eso. Eso no sirve para nada, solo acopla a los tools que deben implementarse y puede que existan pocos o varios.

2.Application

- Refactoriza el context_manager.go. No debe tener nada que ver con models.IntentType, eso no es necesario, solo puede guardar el contenido. Elimina funciones que no son necesarias como buildEntityContext. UpdateIntent tambien. Verifica que es lo que importa de guardar el ultimo intent, que si es valido guardar la tool seleccionada. Si es asi solo guarda como string la tool seleccionada, pero el IntentType debe desaparecer.
- El archivo intent_detector.go debe ser refactorizado, ya no debe usar IntentType. El intent solo debe validar veracidad de intencion, nada mas. Para todos deberia ser puro regex con verbos y validación solo debe haber de listado, creacion, actualizacion. Todos los verbos relacionados a eso. Pero ese "intent" debe ser una implementacion del domain. Deberia llamarse IntentRegex esa implementacion o similar. Porque puede haber otra implementacion donde se le pase el comando del usuario tambiem y que se haga una consulta a un llm para poder determinar si la intencion es correcta.
