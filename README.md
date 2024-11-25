# flinkvertify-友链自动验证API

## 项目介绍
这个程序是用于在他人申请你的网站的友链时，自动验证他是否已经添加你的网站信息到他的网站的友链页面，可以更方便的验证双向友链。

### 功能逻辑
通过API传入需要验证的网站的友链页面的URL，例如“https://example.com/links" ，程序会自动抓取页面并验证页面上是否存在你需要验证的信息。再通过API传入查询请求，即可获得结果。

---

## 使用方法

### 直接使用
1. 下载Releases中的程序到你的服务器

2. 给予运行权限
   `chmod +x ./flinkvertify`

3. 运行程序，替换三个变量
`./flinkvertify -n="你的网站名称" -d="你的网站简介" -p=8080`

4. 程序默认输出示例如下

```
2024/11/25 12:00:00 检测网站名称: example
2024/11/25 12:00:00 检测网站简介: domain
2024/11/25 12:00:00 运行端口: 8080
2024/11/25 12:00:00 Server started with API Key: NHYKURNOMGHTHNVN
```

5. 后台运行
   将程序注册成系统服务，使用systemctl管理，方法自行搜索

### 编译使用
如果你不想用我编译好的程序，请使用**最新版本**的Golang自行搭建Go环境编译

---

## API详情

#### **1. 提交任务**
- **Endpoint**: `/api/task`
- **Method**: `POST`
- **Headers**:
  - `Content-Type: application/json`
  - `X-API-Key`: [程序运行后输出的API KEY]
- **Request Body**:
  ```json
  {
    "url": "https://example.com"
  }
  ```
- **Response**:
  - **成功**:
    ```json
    {
      "task_id": "b2f7412b-1234-5678-9012-abcdef123456"
    }
    ```

---

#### **2. 查询任务结果**
- **Endpoint**: `/api/result`
- **Method**: `GET`
- **Headers**:
  - `X-API-Key`: [程序运行后输出的API KEY]
- **Query Parameters**:
  - `taskid`: 提交任务时返回的 `task_id`
- **Response**:
  - **处理中**:
    ```json
    {
      "status": "processing"
    }
    ```
  - **完成**（关键词存在）:
    ```json
    {
      "result": true,
      "status": "success",
      "task_id": "1732513329707151250"
    }
    ```
  - **完成**（关键词不存在）:
    ```json
    {
      "result": false,
      "status": "success",
      "task_id": "1732513329707151250"
    }
    ```
  - **任务失败**:
    ```json
    {
    "result": false,
    "status": "failure",
    "task_id": "1732519356077725251"
    }
    ```

---

## 开发与贡献

* 欢迎提出改进和建议，有问题请提交Issue

---

## 许可

MIT License.
