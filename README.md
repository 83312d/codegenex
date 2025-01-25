# CodeGenEx

инструмент для генерации сущностей (миграции бд, модели итд)

## Конфигурация

на данный момент файл codegenex.json

## Синтаксис команды

`./codegenex <migration_name> [field:type:options ...]`

### Параметры

1. `<migration_name>`: 
   - Обязательный параметр
   - Определяет имя миграции и соответствующей модели
   - Пример: create_users, add_column_to_posts

2. `[field:type:options ...]`:
   - Необязательный параметр (можно указать несколько)
   - Описывает поля таблицы
   - Формат: имя_поля:тип_поля:опция1:опция2:...

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

`./codegenex create_users name:string:i email:string:unique:i password:string`

`./codegenex create_posts title:string:i content:string user_id:int:ref:i published:bool:default=false views_count:int:default=0`

` ./codegenex create_orders status:enum[pending,processing,completed]:i total:float:i user_id:int:ref:i notes:string:null`

## Примечания

- Имена таблиц автоматически преобразуются во множественное число
- Миграции автоматически содержат таймстэмпы и id
- Имена моделей автоматически преобразуются в CamelCase
- Внешние ключи автоматически создаются для полей, оканчивающихся на _id (временно)
- ENUM типы автоматически создаются в базе данных и представляются как строковые константы в Go
   
