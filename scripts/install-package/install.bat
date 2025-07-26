@echo off
setlocal enabledelayedexpansion

echo ===========================================
echo       Context-Keeper 安装程序 (Windows)
echo ===========================================
echo.

:: 获取脚本所在目录
set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

:: 默认安装目录
set "DEFAULT_INSTALL_DIR=%USERPROFILE%\.context-keeper"

:: 询问安装目录
echo 请输入安装目录 [默认: %DEFAULT_INSTALL_DIR%]:
set /p INSTALL_DIR=
if "!INSTALL_DIR!"=="" set "INSTALL_DIR=%DEFAULT_INSTALL_DIR%"

echo.
echo [步骤] 创建必要的目录...
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"
if not exist "%INSTALL_DIR%\bin" mkdir "%INSTALL_DIR%\bin"
if not exist "%INSTALL_DIR%\config" mkdir "%INSTALL_DIR%\config"
if not exist "%INSTALL_DIR%\data" mkdir "%INSTALL_DIR%\data"
if not exist "%INSTALL_DIR%\logs" mkdir "%INSTALL_DIR%\logs"

echo.
echo [步骤] 复制程序文件...
copy "%SCRIPT_DIR%\bin\context-keeper.exe" "%INSTALL_DIR%\bin\" /Y

echo.
:: 复制配置文件
if exist "%SCRIPT_DIR%\config" (
    echo [步骤] 复制配置文件...
    xcopy "%SCRIPT_DIR%\config\*" "%INSTALL_DIR%\config\" /E /I /Y
)

:: 创建启动脚本
echo.
echo [步骤] 创建启动脚本...
echo @echo off > "%INSTALL_DIR%\start-context-keeper.bat"
echo "%INSTALL_DIR%\bin\context-keeper.exe" >> "%INSTALL_DIR%\start-context-keeper.bat"

echo.
echo [步骤] 验证安装...
echo 运行版本检查:
call "%INSTALL_DIR%\bin\context-keeper.exe" --version
if %ERRORLEVEL% NEQ 0 (
    echo [错误] 无法执行 context-keeper。请确保您有执行权限和所需的系统依赖。
    exit /b 1
)

echo.
echo ===========================================
echo        安装完成！
echo ===========================================
echo Context-Keeper 已安装到 %INSTALL_DIR%
echo.
echo 请在 Cursor 编辑器中配置 MCP 服务器路径为:
echo %INSTALL_DIR%\bin\context-keeper.exe
echo.
echo ===== 使用方法 =====
echo 1. 打开 Cursor 编辑器设置 (Ctrl+,)
echo 2. 搜索 "MCP" 或 "Model Context Protocol"
echo 3. 在 MCP 服务器配置部分添加以下路径:
echo    %INSTALL_DIR%\bin\context-keeper.exe
echo 4. 保存设置并重启 Cursor
echo.
echo 查看更多使用说明，请参考 README.md 文件。
echo.
echo 感谢使用 Context-Keeper！
echo.

pause 