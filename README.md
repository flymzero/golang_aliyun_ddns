# golang_aliyun_ddns
# AccessKeyId  和 AccessKeySecret 根据自己账号填写
# 添加错误提醒(简聊),分组:raspberry

#树莓派开机启动

- 编辑/etc/rc.local文件
-  在**exit 0** 上面一行添加 go run /home/mzero/rpiDdns.go &