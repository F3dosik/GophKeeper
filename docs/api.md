# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [auth.proto](#auth-proto)
    - [CreateUserRequest](#auth-CreateUserRequest)
    - [CreateUserResponse](#auth-CreateUserResponse)
    - [Credentials](#auth-Credentials)
    - [GetSaltRequest](#auth-GetSaltRequest)
    - [GetSaltResponse](#auth-GetSaltResponse)
    - [LoginRequest](#auth-LoginRequest)
    - [LoginResponse](#auth-LoginResponse)
  
    - [Auth](#auth-Auth)
  
- [secrets.proto](#secrets-proto)
    - [CreateSecretRequest](#secrets-CreateSecretRequest)
    - [CreateSecretResponse](#secrets-CreateSecretResponse)
    - [DeleteSecretRequest](#secrets-DeleteSecretRequest)
    - [DeleteSecretResponse](#secrets-DeleteSecretResponse)
    - [GetSecretRequest](#secrets-GetSecretRequest)
    - [GetSecretResponse](#secrets-GetSecretResponse)
    - [ListSecretsRequest](#secrets-ListSecretsRequest)
    - [ListSecretsResponse](#secrets-ListSecretsResponse)
    - [SecretData](#secrets-SecretData)
    - [SecretItem](#secrets-SecretItem)
    - [UpdateSecretRequest](#secrets-UpdateSecretRequest)
    - [UpdateSecretResponse](#secrets-UpdateSecretResponse)
  
    - [Secrets](#secrets-Secrets)
  
- [Scalar Value Types](#scalar-value-types)



<a name="auth-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## auth.proto



<a name="auth-CreateUserRequest"></a>

### CreateUserRequest
CreateUserRequest — запрос регистрации нового пользователя.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| credentials | [Credentials](#auth-Credentials) |  | Учётные данные будущего пользователя. |
| salt | [bytes](#bytes) |  | Случайная соль (16&#43; байт), сгенерированная клиентом, — используется при деривации ключа. |






<a name="auth-CreateUserResponse"></a>

### CreateUserResponse
CreateUserResponse — пустой ответ при успешной регистрации.






<a name="auth-Credentials"></a>

### Credentials
Credentials — учётные данные пользователя, используемые при регистрации и входе.
master_key — производный ключ, вычисленный клиентом из пароля и соли (Argon2id).
Сервер хранит его хеш, сам пароль на сервер не передаётся.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| login | [string](#string) |  | Логин пользователя. |
| master_key | [bytes](#bytes) |  | Производный ключ (32 байта), полученный на клиенте. |






<a name="auth-GetSaltRequest"></a>

### GetSaltRequest
GetSaltRequest — запрос соли пользователя по его логину.
Клиент запрашивает соль перед Login, чтобы получить тот же master_key, что и при регистрации.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| login | [string](#string) |  | Логин пользователя. |






<a name="auth-GetSaltResponse"></a>

### GetSaltResponse
GetSaltResponse — ответ с солью пользователя.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| salt | [bytes](#bytes) |  | Соль, сохранённая при регистрации пользователя. |






<a name="auth-LoginRequest"></a>

### LoginRequest
LoginRequest — запрос на вход в систему.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| credentials | [Credentials](#auth-Credentials) |  | Учётные данные пользователя. |






<a name="auth-LoginResponse"></a>

### LoginResponse
LoginResponse — ответ на успешный вход.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  | JWT-токен, который клиент передаёт в метаданных последующих запросов. |





 

 

 


<a name="auth-Auth"></a>

### Auth
Auth — сервис аутентификации и регистрации пользователей.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateUser | [CreateUserRequest](#auth-CreateUserRequest) | [CreateUserResponse](#auth-CreateUserResponse) | CreateUser регистрирует нового пользователя. Ошибки: AlreadyExists — логин занят; InvalidArgument — невалидные данные. |
| GetSalt | [GetSaltRequest](#auth-GetSaltRequest) | [GetSaltResponse](#auth-GetSaltResponse) | GetSalt возвращает соль пользователя, сохранённую при регистрации. Ошибка: NotFound — пользователь не найден. |
| Login | [LoginRequest](#auth-LoginRequest) | [LoginResponse](#auth-LoginResponse) | Login выполняет вход и возвращает JWT-токен. Ошибки: NotFound — пользователь не найден; Unauthenticated — неверные учётные данные. |

 



<a name="secrets-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## secrets.proto



<a name="secrets-CreateSecretRequest"></a>

### CreateSecretRequest
CreateSecretRequest — запрос создания секрета.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| item | [SecretData](#secrets-SecretData) |  | Создаваемый секрет. |






<a name="secrets-CreateSecretResponse"></a>

### CreateSecretResponse
CreateSecretResponse — пустой ответ при успешном создании.






<a name="secrets-DeleteSecretRequest"></a>

### DeleteSecretRequest
DeleteSecretRequest — запрос удаления секрета.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blind_index | [string](#string) |  | Индекс удаляемого секрета. |






<a name="secrets-DeleteSecretResponse"></a>

### DeleteSecretResponse
DeleteSecretResponse — пустой ответ при успешном удалении.






<a name="secrets-GetSecretRequest"></a>

### GetSecretRequest
GetSecretRequest — запрос на получение секрета по blind_index.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blind_index | [string](#string) |  | Индекс искомого секрета. |






<a name="secrets-GetSecretResponse"></a>

### GetSecretResponse
GetSecretResponse — ответ с зашифрованным секретом и метаданными.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [bytes](#bytes) |  | Зашифрованные данные секрета. |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Дата и время создания. |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Дата и время последнего обновления. |






<a name="secrets-ListSecretsRequest"></a>

### ListSecretsRequest
ListSecretsRequest — пустой запрос списка секретов текущего пользователя.






<a name="secrets-ListSecretsResponse"></a>

### ListSecretsResponse
ListSecretsResponse — ответ со списком секретов.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [SecretItem](#secrets-SecretItem) | repeated | Все секреты текущего пользователя. |






<a name="secrets-SecretData"></a>

### SecretData
SecretData — единица хранения секрета на сервере.
Сервер никогда не видит открытые данные: data зашифрована на клиенте (AES-256-GCM),
blind_index — детерминированный HMAC-SHA256 от (имя, тип) для поиска без раскрытия имени.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blind_index | [string](#string) |  | Детерминированный индекс для поиска секрета (HMAC-SHA256). |
| data | [bytes](#bytes) |  | Зашифрованный полезный payload секрета. |






<a name="secrets-SecretItem"></a>

### SecretItem
SecretItem — запись секрета для списка: зашифрованные данные и метаданные.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blind_index | [string](#string) |  | Детерминированный индекс секрета. |
| data | [bytes](#bytes) |  | Зашифрованные данные секрета. |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Дата и время создания. |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Дата и время последнего обновления. |






<a name="secrets-UpdateSecretRequest"></a>

### UpdateSecretRequest
UpdateSecretRequest — запрос обновления существующего секрета.
Секрет идентифицируется по blind_index.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| item | [SecretData](#secrets-SecretData) |  | Новое содержимое секрета (blind_index остаётся прежним). |






<a name="secrets-UpdateSecretResponse"></a>

### UpdateSecretResponse
UpdateSecretResponse — пустой ответ при успешном обновлении.





 

 

 


<a name="secrets-Secrets"></a>

### Secrets
Secrets — сервис управления зашифрованными секретами пользователя.
Все методы требуют JWT-токен в метаданных (authorization: Bearer ...).

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListSecrets | [ListSecretsRequest](#secrets-ListSecretsRequest) | [ListSecretsResponse](#secrets-ListSecretsResponse) | ListSecrets возвращает все секреты текущего пользователя. |
| CreateSecret | [CreateSecretRequest](#secrets-CreateSecretRequest) | [CreateSecretResponse](#secrets-CreateSecretResponse) | CreateSecret создаёт новый секрет. Ошибка: AlreadyExists — секрет с таким blind_index уже существует. |
| UpdateSecret | [UpdateSecretRequest](#secrets-UpdateSecretRequest) | [UpdateSecretResponse](#secrets-UpdateSecretResponse) | UpdateSecret обновляет существующий секрет. Ошибка: NotFound — секрет с таким blind_index не найден. |
| GetSecret | [GetSecretRequest](#secrets-GetSecretRequest) | [GetSecretResponse](#secrets-GetSecretResponse) | GetSecret возвращает секрет по blind_index. Ошибка: NotFound — секрет не найден. |
| DeleteSecret | [DeleteSecretRequest](#secrets-DeleteSecretRequest) | [DeleteSecretResponse](#secrets-DeleteSecretResponse) | DeleteSecret удаляет секрет по blind_index. Ошибка: NotFound — секрет не найден. |

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

