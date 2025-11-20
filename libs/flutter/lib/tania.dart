// auth_provider.dart
import 'package:flutter/material.dart';
import 'dart:async';

import 'client.dart' show Client;

// ==================== AUTH PROVIDER ====================

/// Provider qui g�re l'�tat d'authentification
class AuthProvider extends InheritedWidget {
  final Client client;
  final Map<String, dynamic>? user;
  final bool isAuthenticated;
  final bool isLoading;
  final Future<void> Function(String email, String password) login;
  final Future<void> Function() logout;
  final Future<void> Function() refreshAuth;

  const AuthProvider({
    required this.client,
    required this.user,
    required this.isAuthenticated,
    required this.isLoading,
    required this.login,
    required this.logout,
    required this.refreshAuth,
    required super.child,
    super.key,
  });

  static AuthProvider? of(BuildContext context) {
    return context.dependOnInheritedWidgetOfExactType<AuthProvider>();
  }

  static AuthProvider ofRequired(BuildContext context) {
    final provider = of(context);
    if (provider == null) {
      throw Exception(
        'AuthProvider not found in context. Wrap your app with AuthProviderWidget.',
      );
    }
    return provider;
  }

  @override
  bool updateShouldNotify(AuthProvider oldWidget) {
    return user != oldWidget.user ||
        isAuthenticated != oldWidget.isAuthenticated ||
        isLoading != oldWidget.isLoading;
  }
}

// ==================== AUTH PROVIDER WIDGET ====================

/// Widget qui g�re l'authentification et fournit le contexte
class AuthProviderWidget extends StatefulWidget {
  final Client client;
  final Widget logged;
  final Widget unlogged;
  final bool autoConnect;
  final Widget? loading;

  const AuthProviderWidget({
    super.key,
    required this.client,
    required this.logged,
    required this.unlogged,
    this.autoConnect = true,
    this.loading,
  });

  @override
  State<AuthProviderWidget> createState() => _AuthProviderWidgetState();
}

class _AuthProviderWidgetState extends State<AuthProviderWidget> {
  Map<String, dynamic>? _user;
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _initAuth();
    _setupListeners();
  }

  Future<void> _initAuth() async {
    try {
      final loggedIn = await widget.client.autoLogin();
      if (loggedIn) {
        setState(() {
          _user = widget.client.authRecord;
        });

        // Auto-connecter les canaux si activ�
        if (widget.autoConnect) {
          await widget.client.connectSSE();
          await widget.client.connectUserRoom();
        }
      }
    } catch (error) {
      debugPrint('Auto-login failed: $error');
    } finally {
      if (mounted) {
        setState(() {
          _isLoading = false;
        });
      }
    }
  }

  void _setupListeners() {
    widget.client.on('auth:login', _handleAuthChange);
    widget.client.on('auth:auto-login', _handleAuthChange);
    widget.client.on('auth:refresh', _handleAuthChange);
    widget.client.on('auth:logout', _handleLogout);
  }

  void _handleAuthChange(dynamic record) {
    if (mounted) {
      setState(() {
        _user = record as Map<String, dynamic>?;
      });
    }
  }

  void _handleLogout(dynamic _) {
    if (mounted) {
      setState(() {
        _user = null;
      });
    }
  }

  Future<void> _login(String email, String password) async {
    await widget.client.login(email, password);

    // Auto-connecter les canaux si activ�
    if (widget.autoConnect) {
      await widget.client.connectSSE();
      await widget.client.connectUserRoom();
    }
  }

  Future<void> _logout() async {
    await widget.client.logout();
  }

  Future<void> _refreshAuth() async {
    await widget.client.refreshAuth();
  }

  @override
  void dispose() {
    widget.client.off('auth:login', _handleAuthChange);
    widget.client.off('auth:auto-login', _handleAuthChange);
    widget.client.off('auth:refresh', _handleAuthChange);
    widget.client.off('auth:logout', _handleLogout);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    // Pendant le chargement
    if (_isLoading) {
      return AuthProvider(
        client: widget.client,
        user: _user,
        isAuthenticated: false,
        isLoading: _isLoading,
        login: _login,
        logout: _logout,
        refreshAuth: _refreshAuth,
        child:
            widget.loading ?? const Center(child: CircularProgressIndicator()),
      );
    }

    // Afficher logged ou unlogged selon l'�tat
    return AuthProvider(
      client: widget.client,
      user: _user,
      isAuthenticated: _user != null,
      isLoading: _isLoading,
      login: _login,
      logout: _logout,
      refreshAuth: _refreshAuth,
      child: _user != null ? widget.logged : widget.unlogged,
    );
  }
}

// ==================== LOGGED WIDGET ====================

/// Widget affiche uniquement si le test est vrai
class ShowIf extends StatelessWidget {
  final Widget child;
  final Widget? fallback;
  final Widget? loading;
  final bool Function(AuthProvider auth) testFn;

  const ShowIf(
    this.testFn, {
    super.key,
    required this.child,
    this.loading,
    this.fallback,
  });

  @override
  Widget build(BuildContext context) {
    final auth = AuthProvider.ofRequired(context);

    if (auth.isLoading) {
      return loading ?? const SizedBox.shrink();
    }

    return testFn(auth) ? child : (loading ?? const SizedBox.shrink());
  }
}

/// Widget affich� uniquement si l'utilisateur est connect�
class ShowIfAuthenticated extends ShowIf {
  ShowIfAuthenticated({
    super.key,
    required super.child,
    super.fallback,
    super.loading,
  }) : super((auth) => auth.isLoading);
}

// ==================== UNLOGGED WIDGET ====================

/// Widget affich� uniquement si l'utilisateur n'est PAS connect�
class ShowIfUnAuthenticated extends ShowIf {
  ShowIfUnAuthenticated({
    Key? key,
    required super.child,
    super.fallback,
    super.loading,
  }) : super((auth) => !auth.isLoading, key: key);
}

// ==================== HELPER FUNCTIONS ====================

/// R�cup�rer le client depuis le contexte
Client useClient(BuildContext context) {
  return AuthProvider.ofRequired(context).client;
}

/// R�cup�rer l'utilisateur depuis le contexte
Map<String, dynamic>? useUser(BuildContext context) {
  return AuthProvider.ofRequired(context).user;
}

/// V�rifier si l'utilisateur est authentifi�
bool useIsAuthenticated(BuildContext context) {
  return AuthProvider.ofRequired(context).isAuthenticated;
}

/// V�rifier si l'authentification est en cours de chargement
bool useIsLoading(BuildContext context) {
  return AuthProvider.ofRequired(context).isLoading;
}

// ==================== EXEMPLE D'UTILISATION ====================

/*
import 'package:flutter/material.dart';

void main() {
  final client = Client(
    serverUrl: 'http://localhost:8090',
    authToken: '',
  );

  runApp(MyApp(client: client));
}

class MyApp extends StatelessWidget {
  final Client client;

  const MyApp({Key? key, required this.client}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Auth Demo',
      theme: ThemeData(
        primarySwatch: Colors.blue,
      ),
      home: AuthProviderWidget(
        client: client,
        autoConnect: true,
        loading: const Scaffold(
          body: Center(child: CircularProgressIndicator()),
        ),
        logged: const DashboardPage(),
        unlogged: const LoginPage(),
      ),
    );
  }
}

// ==================== EXEMPLE SIMPLE ====================

class SimpleExample extends StatelessWidget {
  final Client client;

  const SimpleExample({Key? key, required this.client}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      home: AuthProviderWidget(
        client: client,
        logged: const DashboardScreen(),
        unlogged: const LoginScreen(),
      ),
    );
  }
}

// ==================== HOME PAGE ====================

class HomePage extends StatelessWidget {
  const HomePage({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return AuthProviderWidget(
      client: Client(
        serverUrl: 'http://localhost:8090',
        authToken: '',
      ),
      logged: const DashboardPage(),
      unlogged: const LoginPage(),
      loading: const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      ),
    );
  }
}

// ==================== DASHBOARD PAGE (LOGGED) ====================

class DashboardPage extends StatefulWidget {
  const DashboardPage({Key? key}) : super(key: key);

  @override
  State<DashboardPage> createState() => _DashboardPageState();
}

class _DashboardPageState extends State<DashboardPage> {
  List<Map<String, dynamic>> _notifications = [];
  late Client _client;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _client = useClient(context);
    _setupListeners();
  }

  void _setupListeners() {
    _client.on('notification', (data) {
      if (mounted) {
        setState(() {
          _notifications.add(data as Map<String, dynamic>);
        });
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final user = useUser(context);
    final auth = AuthProvider.ofRequired(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Dashboard'),
        actions: [
          IconButton(
            icon: const Icon(Icons.logout),
            onPressed: () async {
              try {
                await auth.logout();
              } catch (e) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text('Logout failed: $e')),
                );
              }
            },
          ),
        ],
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(16.0),
            child: Text(
              'Welcome, ${user?['name'] ?? user?['email']}!',
              style: Theme.of(context).textTheme.headlineSmall,
            ),
          ),
          Expanded(
            child: ListView(
              children: [
                ListTile(
                  title: Text('Notifications (${_notifications.length})'),
                  subtitle: const Text('Recent notifications'),
                ),
                ..._notifications.map((notif) => ListTile(
                  leading: const Icon(Icons.notifications),
                  title: Text(notif['message'] ?? 'No message'),
                  subtitle: Text(notif['title'] ?? ''),
                )),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

// ==================== LOGIN PAGE (UNLOGGED) ====================

class LoginPage extends StatefulWidget {
  const LoginPage({Key? key}) : super(key: key);

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  final _formKey = GlobalKey<FormState>();
  bool _isLoading = false;
  String? _error;

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _handleLogin() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }

    setState(() {
      _isLoading = true;
      _error = null;
    });

    try {
      final auth = AuthProvider.ofRequired(context);
      await auth.login(_emailController.text, _passwordController.text);
    } catch (e) {
      setState(() {
        _error = e.toString();
      });
    } finally {
      if (mounted) {
        setState(() {
          _isLoading = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Login'),
      ),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24.0),
          child: Form(
            key: _formKey,
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  'Welcome',
                  style: Theme.of(context).textTheme.headlineMedium,
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 32),
                TextFormField(
                  controller: _emailController,
                  decoration: const InputDecoration(
                    labelText: 'Email',
                    border: OutlineInputBorder(),
                    prefixIcon: Icon(Icons.email),
                  ),
                  keyboardType: TextInputType.emailAddress,
                  validator: (value) {
                    if (value == null || value.isEmpty) {
                      return 'Please enter your email';
                    }
                    return null;
                  },
                ),
                const SizedBox(height: 16),
                TextFormField(
                  controller: _passwordController,
                  decoration: const InputDecoration(
                    labelText: 'Password',
                    border: OutlineInputBorder(),
                    prefixIcon: Icon(Icons.lock),
                  ),
                  obscureText: true,
                  validator: (value) {
                    if (value == null || value.isEmpty) {
                      return 'Please enter your password';
                    }
                    return null;
                  },
                ),
                if (_error != null) ...[
                  const SizedBox(height: 16),
                  Text(
                    _error!,
                    style: const TextStyle(color: Colors.red),
                    textAlign: TextAlign.center,
                  ),
                ],
                const SizedBox(height: 24),
                ElevatedButton(
                  onPressed: _isLoading ? null : _handleLogin,
                  style: ElevatedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 16),
                  ),
                  child: _isLoading
                      ? const SizedBox(
                          height: 20,
                          width: 20,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Text('Login'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

// ==================== EXEMPLE AVEC NAVIGATION ====================

class AppWithNavigation extends StatelessWidget {
  final Client client;

  const AppWithNavigation({Key? key, required this.client}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Auth Demo',
      theme: ThemeData(primarySwatch: Colors.blue),
      home: AuthProviderWidget(
        client: client,
        logged: const AuthenticatedApp(),
        unlogged: const UnauthenticatedApp(),
        loading: const Scaffold(
          body: Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                CircularProgressIndicator(),
                SizedBox(height: 16),
                Text('Checking authentication...'),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class AuthenticatedApp extends StatelessWidget {
  const AuthenticatedApp({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return const DashboardScreen();
  }
}

class UnauthenticatedApp extends StatelessWidget {
  const UnauthenticatedApp({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return const LoginScreen();
  }
}

// ==================== COMPOSANTS UTILITAIRES ====================

class LogoutButton extends StatefulWidget {
  const LogoutButton({Key? key}) : super(key: key);

  @override
  State<LogoutButton> createState() => _LogoutButtonState();
}

class _LogoutButtonState extends State<LogoutButton> {
  bool _isLoading = false;

  @override
  Widget build(BuildContext context) {
    final auth = AuthProvider.ofRequired(context);

    return ElevatedButton(
      onPressed: _isLoading
          ? null
          : () async {
              setState(() => _isLoading = true);
              try {
                await auth.logout();
              } catch (e) {
                if (mounted) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(content: Text('Logout failed: $e')),
                  );
                }
              } finally {
                if (mounted) {
                  setState(() => _isLoading = false);
                }
              }
            },
      child: _isLoading
          ? const SizedBox(
              height: 16,
              width: 16,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : const Text('Logout'),
    );
  }
}

class UserProfile extends StatelessWidget {
  const UserProfile({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final user = useUser(context);
    final auth = AuthProvider.ofRequired(context);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Profile', style: Theme.of(context).textTheme.headlineSmall),
            const SizedBox(height: 16),
            Text('Email: ${user?['email']}'),
            Text('Name: ${user?['name'] ?? 'N/A'}'),
            Text('Username: ${user?['username'] ?? 'N/A'}'),
            const SizedBox(height: 16),
            ElevatedButton(
              onPressed: () async {
                try {
                  await auth.refreshAuth();
                  if (context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('Token refreshed!')),
                    );
                  }
                } catch (e) {
                  if (context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(content: Text('Refresh failed: $e')),
                    );
                  }
                }
              },
              child: const Text('Refresh Token'),
            ),
          ],
        ),
      ),
    );
  }
}

class ConnectionStatus extends StatefulWidget {
  const ConnectionStatus({Key? key}) : super(key: key);

  @override
  State<ConnectionStatus> createState() => _ConnectionStatusState();
}

class _ConnectionStatusState extends State<ConnectionStatus> {
  bool _sseConnected = false;
  bool _roomConnected = false;

  @override
  void initState() {
    super.initState();
    _setupListeners();
  }

  void _setupListeners() {
    final client = useClient(context);

    client.on('sse:connected', (_) {
      if (mounted) setState(() => _sseConnected = true);
    });

    client.on('sse:disconnected', (_) {
      if (mounted) setState(() => _sseConnected = false);
    });

    client.on('room:connected', (_) {
      if (mounted) setState(() => _roomConnected = true);
    });

    client.on('room:disconnected', (_) {
      if (mounted) setState(() => _roomConnected = false);
    });
  }

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        _StatusIndicator(
          label: 'SSE',
          isConnected: _sseConnected,
        ),
        const SizedBox(width: 16),
        _StatusIndicator(
          label: 'WebRTC',
          isConnected: _roomConnected,
        ),
      ],
    );
  }
}

class _StatusIndicator extends StatelessWidget {
  final String label;
  final bool isConnected;

  const _StatusIndicator({
    required this.label,
    required this.isConnected,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(label),
        const SizedBox(width: 4),
        Icon(
          isConnected ? Icons.check_circle : Icons.cancel,
          color: isConnected ? Colors.green : Colors.red,
          size: 16,
        ),
      ],
    );
  }
}

// ==================== SCREENS ====================

class DashboardScreen extends StatelessWidget {
  const DashboardScreen({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final user = useUser(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Dashboard'),
        actions: const [
          Padding(
            padding: EdgeInsets.all(8.0),
            child: ConnectionStatus(),
          ),
          Padding(
            padding: EdgeInsets.all(8.0),
            child: LogoutButton(),
          ),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'Welcome, ${user?['name'] ?? user?['email']}!',
              style: Theme.of(context).textTheme.headlineMedium,
            ),
            const SizedBox(height: 24),
            const UserProfile(),
            const SizedBox(height: 24),
            const NotificationsList(),
          ],
        ),
      ),
    );
  }
}

class LoginScreen extends StatelessWidget {
  const LoginScreen({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Login')),
      body: const LoginPage(),
    );
  }
}

class NotificationsList extends StatefulWidget {
  const NotificationsList({Key? key}) : super(key: key);

  @override
  State<NotificationsList> createState() => _NotificationsListState();
}

class _NotificationsListState extends State<NotificationsList> {
  final List<Map<String, dynamic>> _notifications = [];

  @override
  void initState() {
    super.initState();
    final client = useClient(context);

    client.on('notification', (data) {
      if (mounted) {
        setState(() {
          _notifications.insert(0, data as Map<String, dynamic>);
        });
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Notifications (${_notifications.length})',
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 16),
            if (_notifications.isEmpty)
              const Text('No notifications yet')
            else
              ..._notifications.map((notif) => ListTile(
                leading: const Icon(Icons.notifications),
                title: Text(notif['message'] ?? 'No message'),
                subtitle: Text(notif['title'] ?? ''),
              )),
          ],
        ),
      ),
    );
  }
}
*/
