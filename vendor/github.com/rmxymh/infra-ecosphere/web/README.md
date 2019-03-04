# WEB Service

This package provides Web APIs to control BMC operations. The original purpose is to make a VM where the provisioning service is located to call this application in host OS directly.

By default, Web APIs can be access via HTTP with port 9090.

## APIs

Currently, we supports the following APIs with JSON data as their context types:

* GET /api/BMCs
    * Get all BMC information
* GET /api/BMCs/<BMC_IP>
    * Get the information of the specified BMC
* PUT /api/BMCs/<BMC_IP>/power
    * Send power operation to the BMC 
* PUT /api/BMCs/<BMC_IP>/bootdev
    * Set boot device to the BMC

More information can be refer to the following sessions

### GET /api/BMCs
* Description: Get all BMC information
* Request Body: NONE
* Response Example:

```json
{
    "BMCs": [
        {
            "IP": "127.0.1.1",
            "PowerStatus": "ON"
        },
        {
            "IP": "127.0.1.2",
            "PowerStatus": "OFF"
        },
        {
            "IP": "127.0.1.3",
            "PowerStatus": "ON"
        }
    ]
}
```

* Response Data Fields:
    * BMCs: A list contains all BMC information.
        * IP: BMC IP Address
        * PowerStatus: Current power status. (ON / OFF)

### GET /api/BMCs/<BMC_IP>
* Description: Get the information of the specified BMC
* Request Body: NONE
* Response Example:

```json
{
    "IP": "127.0.1.1",
    "PowerStatus": "ON"
}
```

* Response Data Fields:
    * IP: BMC IP Address
    * PowerStatus: Current power status. (ON / OFF)

### PUT /api/BMCs/{BMC_IP}/power
* Description: Send power operation to the BMC
* Request Body:

```json
{
    "Operation": <POWER_OPERATION>
}
```

* Request Body Fields:
    * Operation: Power operation (ON / OFF / RESET / CYCLE)
* Response Example:

```json
{
    "IP": "127.0.1.1",
    "Operation": "OFF",
    "Status": "OK"
}
```

* Response Data Fields:
    * IP: BMC IP Address
    * Operation: The power operation we want to perform.
    * Status: Operation result
    
* Note: It may take some time to make power operation effect. After this API is called, you can use GET /api/BMCs/<BMC_IP> to fetch the current power states.

 
### PUT /api/BMCs/{BMC_IP}/bootdev
* Description: Set boot device to the BMC
* Request Body:

```json
{
    "Device": <BOOT_DEVICE_NAME>
}
```

* Request Body Fields:
    * Device: Device name which we want to specify. ( DISK / PXE )
* Response Example:

```json
{
    "IP": "127.0.1.1",
    "Device": "DISK",
    "Status": "OK"
}
```

* Response Data Fields:
    * IP: BMC IP Address
    * Device: The boot device value we want to set.
    * Status: Operation result 

## Reference

All the Restful API Web Server implementation idea is from [Making a RESTful JSON API in Go](http://thenewstack.io/make-a-restful-json-api-go/).

In this package, routes and logger implementation are based on this tutorial.