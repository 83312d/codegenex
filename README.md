# CodeGenEx

инструмент для генерации сущностей (миграции бд, модели итд)

## Конфигурация

на данный момент файл codegenex.json

## Синтаксис команды

`./codegenex <entity_name> <action> [field:type:options ...]`

### Параметры

1. `<entity_name>`: 
   - Обязательный параметр
   - Определяет имя сущности с которой производится операция, соответвует названию таблицы
   - Пример: users, products

2. `<action>`:
   - Обязательный параметр
   - Определяет тип действия которое будет произведено над сущностью

3. `[field:type:options ...]`:
   - Необязательный параметр (можно указать несколько)
   - Описывает поля таблицы
   - Формат: имя_поля:тип_поля:опция1:опция2:...

### Действия
- `create`: создание сущности
- `add_fields`: добавление к сущности полей(в том числе связей)
- `remove_fields`: удаление полей из сущности
- `drop`: удаление сущности

### Типы полей

- `int`: целое число (в Go: int64, в SQL: INTEGER)
- `string`: строка (в Go: string, в SQL: VARCHAR(255))
- `bool`: булево значение (в Go: bool, в SQL: BOOLEAN)
- `time`: временная метка (в Go: time.Time, в SQL: TIMESTAMP)
- `float`: число с плавающей точкой (в Go: float64, в SQL: NUMERIC)
- `jsonb`: JSON данные (в Go: map[string]interface{}, в SQL: JSONB)
- `enum[value1,value2,...]`: перечисление (в Go: string константы, в SQL: ENUM)

### Опции полей

- `i`: создать индекс для этого поля
- `unique`: поле должно быть уникальным
- `null`: поле может быть NULL
- `default=value`: установить значение по умолчанию
- ref`: поле является внешним ключом (для отношений между таблицами)
- `ref=option`: указать опцию для внешнего ключа (cascade, nullify, restrict, no_action)

### Примеры вызова

`./codegenex users create name:string:i email:string:unique:i password:string`

`./codegenex posts create title:string:i content:string user_id:int:ref:i published:bool:default=false views_count:int:default=0`

` ./codegenex orders create status:enum[pending,processing,completed]:i total:float:i user_id:int:ref:i notes:string:null`

`./codegenex users add_fields middle_name:string last_name:string:unique`

`./codegenex users remove_fields middle_name:string`

`./codegenex users drop`

## Примечания

- Имена таблиц автоматически преобразуются во множественное число
- Миграции автоматически содержат таймстэмпы и id
- Имена моделей автоматически преобразуются в CamelCase
- Внешние ключи автоматически создаются для полей, оканчивающихся на _id (временно)
- ENUM типы автоматически создаются в базе данных и представляются как строковые константы в Go
   
