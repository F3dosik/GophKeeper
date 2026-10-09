# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [admin.proto](#admin-proto)
    - [DeleteUserRequest](#admin-DeleteUserRequest)
    - [DeleteUserResponse](#admin-DeleteUserResponse)
    - [GetUserRequest](#admin-GetUserRequest)
    - [GetUserResponse](#admin-GetUserResponse)
    - [ListUsersRequest](#admin-ListUsersRequest)
    - [ListUsersResponse](#admin-ListUsersResponse)
    - [RevokeSessionsRequest](#admin-RevokeSessionsRequest)
    - [RevokeSessionsResponse](#admin-RevokeSessionsResponse)
    - [UserInfo](#admin-UserInfo)
  
    - [Admin](#admin-Admin)
  
- [auth.proto](#auth-proto)
    - [ChangePasswordHeader](#auth-ChangePasswordHeader)
    - [ChangePasswordRequest](#auth-ChangePasswordRequest)
    - [ChangePasswordResponse](#auth-ChangePasswordResponse)
    - [CreateUserRequest](#auth-CreateUserRequest)
    - [CreateUserResponse](#auth-CreateUserResponse)
    - [Credentials](#auth-Credentials)
    - [DeleteAccountRequest](#auth-DeleteAccountRequest)
    - [DeleteAccountResponse](#auth-DeleteAccountResponse)
    - [GetSaltRequest](#auth-GetSaltRequest)
    - [GetSaltResponse](#auth-GetSaltResponse)
    - [KDFParams](#auth-KDFParams)
    - [LoginRequest](#auth-LoginRequest)
    - [LoginResponse](#auth-LoginResponse)
    - [LogoutRequest](#auth-LogoutRequest)
    - [LogoutResponse](#auth-LogoutResponse)
    - [ReencryptedSecret](#auth-ReencryptedSecret)
  
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



<a name="admin-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## admin.proto



<a name="admin-DeleteUserRequest"></a>

### DeleteUserRequest
DeleteUserRequest — запрос удаления пользователя администратором.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| login | [string](#string) |  | Логин удаляемого пользователя. |






<a name="admin-DeleteUserResponse"></a>

### DeleteUserResponse
DeleteUserResponse — пустой ответ при успешном удалении.






<a name="admin-GetUserRequest"></a>

### GetUserRequest
GetUserRequest — запрос сведений о пользователе.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| login | [string](#string) |  |  |






<a name="admin-GetUserResponse"></a>

### GetUserResponse
GetUserResponse — сведения о пользователе.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [UserInfo](#admin-UserInfo) |  |  |






<a name="admin-ListUsersRequest"></a>

### ListUsersRequest
ListUsersRequest — запрос страницы списка пользователей.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Максимум пользователей на странице (1–100, по умолчанию 100). |
| page_token | [string](#string) |  | Курсор из next_page_token; пустой — первая страница. |






<a name="admin-ListUsersResponse"></a>

### ListUsersResponse
ListUsersResponse — страница списка пользователей, по возрастанию логина.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| users | [UserInfo](#admin-UserInfo) | repeated |  |
| next_page_token | [string](#string) |  | Курсор следующей страницы; пустой, если страница последняя. |






<a name="admin-RevokeSessionsRequest"></a>

### RevokeSessionsRequest
RevokeSessionsRequest — запрос завершения всех сессий пользователя.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| login | [string](#string) |  |  |






<a name="admin-RevokeSessionsResponse"></a>

### RevokeSessionsResponse
RevokeSessionsResponse — пустой ответ.






<a name="admin-UserInfo"></a>

### UserInfo
UserInfo — сведения о пользователе для администратора. Содержимое секретов
администратору недоступно: оно зашифровано ключом пользователя.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| login | [string](#string) |  | Логин. |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Время регистрации. |
| secret_count | [int32](#int32) |  | Число секретов. |
| temporary_password_expires_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Для временного пароля — время, до которого им можно войти; не задано, если пароль постоянный. |
| kdf_time | [uint32](#uint32) |  | Параметры Argon2id пользователя: проходы и память в КиБ. |
| kdf_memory_kib | [uint32](#uint32) |  |  |





 

 

 


<a name="admin-Admin"></a>

### Admin
Admin — административный сервис. Доступен только на административном порту
сервера (ADMIN_PORT), который публикуется на 127.0.0.1; на публичном порту
сервис не зарегистрирован. Методы не требуют токена: доступ ограничен сетью.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| DeleteUser | [DeleteUserRequest](#admin-DeleteUserRequest) | [DeleteUserResponse](#admin-DeleteUserResponse) | DeleteUser безвозвратно удаляет пользователя вместе со всеми секретами; все его токены перестают действовать. Ошибки: NotFound — пользователь не найден; InvalidArgument — неверный логин. |
| ListUsers | [ListUsersRequest](#admin-ListUsersRequest) | [ListUsersResponse](#admin-ListUsersResponse) | ListUsers возвращает страницу списка пользователей. Ошибка: InvalidArgument — неверный page_size или page_token. |
| GetUser | [GetUserRequest](#admin-GetUserRequest) | [GetUserResponse](#admin-GetUserResponse) | GetUser возвращает сведения о пользователе. Ошибка: NotFound — пользователь не найден. |
| RevokeSessions | [RevokeSessionsRequest](#admin-RevokeSessionsRequest) | [RevokeSessionsResponse](#admin-RevokeSessionsResponse) | RevokeSessions отзывает все токены пользователя: на всех устройствах потребуется войти заново. Пароль и секреты не меняются. Ошибка: NotFound — пользователь не найден. |

 



<a name="auth-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## auth.proto



<a name="auth-ChangePasswordHeader"></a>

### ChangePasswordHeader
ChangePasswordHeader — первое сообщение потока смены пароля.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| old_auth_key | [bytes](#bytes) |  | Ключ аутентификации от текущего пароля: подтверждает, что пароль знает владелец, а не только тот, у кого есть токен. |
| new_salt | [bytes](#bytes) |  | Новая соль (16 байт). |
| new_auth_key | [bytes](#bytes) |  | Ключ аутентификации от нового пароля (32 байта). |
| new_kdf | [KDFParams](#auth-KDFParams) |  | Параметры Argon2id, с которыми выведен новый ключ. |






<a name="auth-ChangePasswordRequest"></a>

### ChangePasswordRequest
ChangePasswordRequest — сообщение потока смены пароля: сначала header,
затем по одному secret на каждый секрет пользователя.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| header | [ChangePasswordHeader](#auth-ChangePasswordHeader) |  |  |
| secret | [ReencryptedSecret](#auth-ReencryptedSecret) |  |  |






<a name="auth-ChangePasswordResponse"></a>

### ChangePasswordResponse
ChangePasswordResponse — ответ на успешную смену пароля.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  | Новый токен. Все ранее выданные токены пользователя отозваны. |






<a name="auth-CreateUserRequest"></a>

### CreateUserRequest
CreateUserRequest — запрос регистрации нового пользователя.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| credentials | [Credentials](#auth-Credentials) |  | Учётные данные будущего пользователя. |
| salt | [bytes](#bytes) |  | Случайная соль (16 байт), сгенерированная клиентом, — используется при деривации ключа. |
| kdf | [KDFParams](#auth-KDFParams) |  | Параметры Argon2id, с которыми клиент вывел ключ. Обязательны. |
| temporary | [bool](#bool) |  | true — пароль временный: годен ограниченное время и должен быть сменён при первом входе. Разрешено только на административном порту. |






<a name="auth-CreateUserResponse"></a>

### CreateUserResponse
CreateUserResponse — ответ при успешной регистрации.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| temporary_expires_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Для временного пароля — время, до которого им можно войти. |






<a name="auth-Credentials"></a>

### Credentials
Credentials — учётные данные пользователя, используемые при регистрации и входе.
auth_key — ключ аутентификации: HKDF(masterKey, &#34;auth&#34;), где masterKey = Argon2id(пароль, соль).
Он независим от ключей шифрования секретов, поэтому сервер (и тот, кто прочитает его БД)
не может расшифровать данные. Сервер хранит SHA-256 от auth_key; ни пароль, ни masterKey
на сервер не передаются.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| login | [string](#string) |  | Логин пользователя. |
| auth_key | [bytes](#bytes) |  | Ключ аутентификации (32 байта), полученный на клиенте. |






<a name="auth-DeleteAccountRequest"></a>

### DeleteAccountRequest
DeleteAccountRequest — запрос удаления своей учётки.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| auth_key | [bytes](#bytes) |  | Ключ аутентификации от текущего пароля: одного токена для удаления недостаточно. |






<a name="auth-DeleteAccountResponse"></a>

### DeleteAccountResponse
DeleteAccountResponse — пустой ответ при успешном удалении.






<a name="auth-GetSaltRequest"></a>

### GetSaltRequest
GetSaltRequest — запрос соли пользователя по его логину.
Клиент запрашивает соль перед Login, чтобы получить тот же auth_key, что и при регистрации.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| login | [string](#string) |  | Логин пользователя. |






<a name="auth-GetSaltResponse"></a>

### GetSaltResponse
GetSaltResponse — ответ с солью пользователя.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| salt | [bytes](#bytes) |  | Соль, сохранённая при регистрации пользователя. |
| kdf | [KDFParams](#auth-KDFParams) |  | Параметры Argon2id пользователя. |






<a name="auth-KDFParams"></a>

### KDFParams
KDFParams — параметры Argon2id, с которыми из пароля выводится мастер-ключ.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| time | [uint32](#uint32) |  | Число проходов (1–10). |
| memory_kib | [uint32](#uint32) |  | Объём памяти в КиБ (19456–1048576). |
| threads | [uint32](#uint32) |  | Степень параллелизма (1–16). |






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
| password_change_required | [bool](#bool) |  | true — пароль временный и должен быть сменён. Токен в этом случае короткоживущий и разрешает только ChangePassword и Logout. |






<a name="auth-LogoutRequest"></a>

### LogoutRequest
LogoutRequest — запрос на выход.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| all_sessions | [bool](#bool) |  | true — отозвать все токены пользователя (выйти на всех устройствах), false — только токен, с которым выполнен запрос. |






<a name="auth-LogoutResponse"></a>

### LogoutResponse
LogoutResponse — пустой ответ при успешном выходе.






<a name="auth-ReencryptedSecret"></a>

### ReencryptedSecret
ReencryptedSecret — секрет, перешифрованный ключом от нового пароля.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| old_blind_index | [string](#string) |  | Blind index секрета, вычисленный ключом от текущего пароля. |
| new_blind_index | [string](#string) |  | Blind index, вычисленный ключом от нового пароля. |
| data | [bytes](#bytes) |  | Данные, зашифрованные ключом от нового пароля. |
| expected_updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | updated_at секрета на момент чтения клиентом: если секрет изменили позже, смена пароля отменяется, чтобы не затереть изменение старыми данными. |





 

 

 


<a name="auth-Auth"></a>

### Auth
Auth — сервис аутентификации и регистрации пользователей.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateUser | [CreateUserRequest](#auth-CreateUserRequest) | [CreateUserResponse](#auth-CreateUserResponse) | CreateUser регистрирует нового пользователя. Ошибки: AlreadyExists — логин занят; InvalidArgument — невалидные данные; PermissionDenied — регистрация на этом порту отключена. |
| GetSalt | [GetSaltRequest](#auth-GetSaltRequest) | [GetSaltResponse](#auth-GetSaltResponse) | GetSalt возвращает соль пользователя, сохранённую при регистрации. Ошибка: NotFound — пользователь не найден. |
| Login | [LoginRequest](#auth-LoginRequest) | [LoginResponse](#auth-LoginResponse) | Login выполняет вход и возвращает JWT-токен. Ошибки: NotFound — пользователь не найден; Unauthenticated — неверные учётные данные. |
| Logout | [LogoutRequest](#auth-LogoutRequest) | [LogoutResponse](#auth-LogoutResponse) | Logout отзывает текущий токен или все токены пользователя. Требует JWT-токен в метаданных (authorization: Bearer ...). Ошибка: Unauthenticated — токен отсутствует, истёк или уже отозван. |
| ChangePassword | [ChangePasswordRequest](#auth-ChangePasswordRequest) stream | [ChangePasswordResponse](#auth-ChangePasswordResponse) | ChangePassword меняет пароль и перешифровывает все секреты в одной транзакции: при любой ошибке ничего не меняется. Клиент передаёт header, затем все свои секреты. Требует JWT-токен. После успеха все прежние токены пользователя отозваны. Ошибки: Unauthenticated — неверный текущий пароль или токен; Aborted — секреты изменились во время смены (нужно повторить); InvalidArgument — некорректные данные. |
| DeleteAccount | [DeleteAccountRequest](#auth-DeleteAccountRequest) | [DeleteAccountResponse](#auth-DeleteAccountResponse) | DeleteAccount безвозвратно удаляет учётку текущего пользователя вместе со всеми секретами. Требует JWT-токен и ключ аутентификации от текущего пароля. Ошибка: Unauthenticated — неверный пароль или токен. |

 



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
ListSecretsRequest — запрос страницы списка секретов текущего пользователя.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Максимум секретов на странице (1–100, по умолчанию 100). Сервер может вернуть меньше, чтобы ответ не превышал 4 MiB, но не меньше одного секрета. |
| page_token | [string](#string) |  | Курсор из next_page_token предыдущего ответа; пустой — первая страница. |






<a name="secrets-ListSecretsResponse"></a>

### ListSecretsResponse
ListSecretsResponse — страница списка секретов.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [SecretItem](#secrets-SecretItem) | repeated | Секреты текущей страницы. |
| next_page_token | [string](#string) |  | Курсор следующей страницы; пустой, если страница последняя. |






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
| ListSecrets | [ListSecretsRequest](#secrets-ListSecretsRequest) | [ListSecretsResponse](#secrets-ListSecretsResponse) | ListSecrets возвращает страницу секретов текущего пользователя. Ошибка: InvalidArgument — неверный page_size или page_token. |
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

