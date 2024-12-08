# ecs_agent

守护进程

```
curl -L https://raw.githubusercontent.com/spiritLHLS/ecs_agent/main/ecsagent.sh -o ecsagent.sh && chmod +x ecsagent.sh && bash ecsagent.sh
```

```
systemctl status ecsagent.service
```

```
systemctl stop ecsagent.service
systemctl disable ecsagent.service
systemctl remove ecsagent.service
```

仅测试

```
rm -rf ecsagent
wget https://raw.githubusercontent.com/spiritLHLS/ecs_agent/main/ecsagent
chmod 777 ecsagent
ls
```
