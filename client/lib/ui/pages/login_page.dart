import 'package:flutter/material.dart';
import 'package:flutter/scheduler.dart' show timeDilation;
import 'package:social_media_app/app/configs/colors.dart';
import 'package:social_media_app/app/configs/theme.dart';
import 'package:tania/tania.dart';

class LoginPage extends StatefulWidget {
  const LoginPage({super.key});

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> with TickerProviderStateMixin {
  late AnimationController _animationController;
  late Animation<double> _animation;

  final _formKey = GlobalKey<FormState>();

  // Login controllers
  final _loginEmailController = TextEditingController();
  final _loginPasswordController = TextEditingController();

  // Register controllers
  final _registerNameController = TextEditingController();
  final _registerGenreController = TextEditingController();
  final _registerEmailController = TextEditingController();
  final _registerPhoneController = TextEditingController();
  final _registerUsernameController = TextEditingController();
  final _registerPasswordController = TextEditingController();

  bool _isLogin = true;
  bool _isLoading = false;
  String? _errorMessage;

  @override
  void initState() {
    super.initState();
    _animationController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 500),
    );
    _animation = CurvedAnimation(
      parent: _animationController,
      curve: Curves.easeOut,
      reverseCurve: Curves.easeIn,
    );
    timeDilation = 1.0;
  }

  @override
  void dispose() {
    _animationController.dispose();
    _loginEmailController.dispose();
    _loginPasswordController.dispose();
    _registerNameController.dispose();
    _registerGenreController.dispose();
    _registerEmailController.dispose();
    _registerPhoneController.dispose();
    _registerUsernameController.dispose();
    _registerPasswordController.dispose();
    super.dispose();
  }

  void _toggleForm() {
    setState(() {
      _isLogin = !_isLogin;
      if (_isLogin) {
        _animationController.reverse();
      } else {
        _animationController.forward();
      }
      _errorMessage = null;
    });
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }

    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });

    final auth = AuthProvider.ofRequired(context);

    try {
      if (_isLogin) {
        await auth.login(
          _loginEmailController.text,
          _loginPasswordController.text,
        );
      } else {
        await auth.client.register(
          name: _registerNameController.text,
          genre: _registerGenreController.text,
          email: _registerEmailController.text,
          phone: _registerPhoneController.text,
          username: _registerUsernameController.text,
          password: _registerPasswordController.text,
        );
        // After successful registration, show a message and switch to login view
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
                content:
                    Text('Inscription réussie ! Veuillez vous connecter.')),
          );
          _toggleForm();
        }
      }
    } catch (e) {
      setState(() {
        _errorMessage = e.toString().replaceFirst('Exception: ', '');
      });
    } finally {
      if (mounted) {
        setState(() {
          _isLoading = false;
        });
      }
    }
  }

  Widget _buildTextField({
    required TextEditingController controller,
    required String labelText,
    required IconData icon,
    bool obscureText = false,
    String? Function(String?)? validator,
    TextInputType? keyboardType,
  }) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8.0),
      child: TextFormField(
        controller: controller,
        obscureText: obscureText,
        keyboardType: keyboardType,
        decoration: InputDecoration(
          labelText: labelText,
          prefixIcon: Icon(icon),
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
          ),
        ),
        validator: validator,
      ),
    );
  }

  Widget _buildLoginForm() {
    return Column(
      children: [
        _buildTextField(
          controller: _loginEmailController,
          labelText: 'Email ou Nom d\'utilisateur',
          icon: Icons.person,
          validator: (value) =>
              value!.isEmpty ? 'Veuillez entrer votre email' : null,
        ),
        _buildTextField(
          controller: _loginPasswordController,
          labelText: 'Mot de passe',
          icon: Icons.lock,
          obscureText: true,
          validator: (value) =>
              value!.isEmpty ? 'Veuillez entrer votre mot de passe' : null,
        ),
        const SizedBox(height: 10),
        Align(
          alignment: Alignment.centerRight,
          child: TextButton(
            onPressed: () {
              // TODO: Implement forgot password
            },
            child: const Text('Mot de passe oublié ?'),
          ),
        ),
      ],
    );
  }

  Widget _buildRegisterForm() {
    return Column(
      children: [
        _buildTextField(
          controller: _registerNameController,
          labelText: 'Nom complet',
          icon: Icons.person_outline,
          validator: (value) =>
              value!.isEmpty ? 'Veuillez entrer votre nom' : null,
        ),
        _buildTextField(
          controller: _registerUsernameController,
          labelText: 'Nom d\'utilisateur',
          icon: Icons.alternate_email,
          validator: (value) =>
              value!.isEmpty ? 'Veuillez entrer un nom d\'utilisateur' : null,
        ),
        _buildTextField(
          controller: _registerEmailController,
          labelText: 'Email',
          icon: Icons.email_outlined,
          keyboardType: TextInputType.emailAddress,
          validator: (value) {
            if (value == null || value.isEmpty) {
              return 'Veuillez entrer un email';
            }
            if (!RegExp(r'\S+@\S+\.\S+').hasMatch(value)) {
              return 'Veuillez entrer un email valide';
            }
            return null;
          },
        ),
        _buildTextField(
          controller: _registerPasswordController,
          labelText: 'Mot de passe',
          icon: Icons.lock_outline,
          obscureText: true,
          validator: (value) {
            if (value == null || value.isEmpty) {
              return 'Veuillez entrer un mot de passe';
            }
            if (value.length < 8) {
              return 'Le mot de passe doit faire au moins 8 caractères';
            }
            return null;
          },
        ),
        _buildTextField(
          controller: _registerPhoneController,
          labelText: 'Téléphone (Optionnel)',
          icon: Icons.phone_outlined,
          keyboardType: TextInputType.phone,
        ),
        _buildTextField(
          controller: _registerGenreController,
          labelText: 'Genre (Optionnel)',
          icon: Icons.wc_outlined,
        ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    final deviceSize = MediaQuery.of(context).size;
    final cardWidth = deviceSize.width * 0.85;

    return Scaffold(
      backgroundColor: AppColors.backgroundColor,
      body: Center(
        child: SingleChildScrollView(
          child: SizedBox(
            height: deviceSize.height,
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                // Logo or App Name
                Text(
                  'Tania',
                  style: AppTheme.blackTextStyle.copyWith(
                    fontSize: 40,
                    fontWeight: AppTheme.bold,
                    color: AppColors.primaryColor,
                  ),
                ),
                const SizedBox(height: 30),
                Card(
                  elevation: 8.0,
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(10.0),
                  ),
                  child: AnimatedContainer(
                    duration: const Duration(milliseconds: 300),
                    curve: Curves.easeIn,
                    width: cardWidth,
                    padding: const EdgeInsets.all(24.0),
                    child: Form(
                      key: _formKey,
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          // Animated Cross-fade for forms
                          AnimatedCrossFade(
                            firstChild: _buildLoginForm(),
                            secondChild: _buildRegisterForm(),
                            crossFadeState: _isLogin
                                ? CrossFadeState.showFirst
                                : CrossFadeState.showSecond,
                            duration: const Duration(milliseconds: 300),
                          ),
                          const SizedBox(height: 20),
                          if (_errorMessage != null)
                            Padding(
                              padding: const EdgeInsets.only(bottom: 10),
                              child: Text(
                                _errorMessage!,
                                style: const TextStyle(
                                    color: Colors.red, fontSize: 12),
                                textAlign: TextAlign.center,
                              ),
                            ),
                          _isLoading
                              ? const CircularProgressIndicator()
                              : SizedBox(
                                  width: double.infinity,
                                  child: ElevatedButton(
                                    onPressed: _submit,
                                    style: ElevatedButton.styleFrom(
                                      padding: const EdgeInsets.symmetric(
                                          vertical: 16),
                                      backgroundColor: AppColors.primaryColor,
                                      foregroundColor: AppColors.whiteColor,
                                      shape: RoundedRectangleBorder(
                                        borderRadius:
                                            BorderRadius.circular(8.0),
                                      ),
                                    ),
                                    child: Text(
                                      _isLogin ? 'CONNEXION' : 'INSCRIPTION',
                                      style: AppTheme.whiteTextStyle.copyWith(
                                        fontWeight: AppTheme.bold,
                                      ),
                                    ),
                                  ),
                                ),
                          const SizedBox(height: 20),
                          TextButton(
                            onPressed: _toggleForm,
                            child: Text(
                              _isLogin
                                  ? 'Pas de compte ? S\'inscrire'
                                  : 'Déjà un compte ? Se connecter',
                            ),
                          ),
                          const SizedBox(height: 20),
                          _buildSocialButtons(),
                        ],
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildSocialButtons() {
    return Column(
      children: [
        const Row(
          children: [
            Expanded(child: Divider()),
            Padding(
              padding: EdgeInsets.symmetric(horizontal: 8.0),
              child: Text('OU'),
            ),
            Expanded(child: Divider()),
          ],
        ),
        const SizedBox(height: 20),
        Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            _socialButton(
              'assets/images/ic_google.png', // Assurez-vous que cet asset existe
              () {
                // TODO: Implémenter la connexion Google
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(
                      content: Text('Connexion Google non implémentée')),
                );
              },
            ),
            const SizedBox(width: 20),
            _socialButton(
              'assets/images/ic_facebook.png', // Assurez-vous que cet asset existe
              () {
                // TODO: Implémenter la connexion Facebook
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(
                      content: Text('Connexion Facebook non implémentée')),
                );
              },
            ),
          ],
        ),
      ],
    );
  }

  Widget _socialButton(String assetName, VoidCallback onPressed) {
    return InkWell(
      onTap: onPressed,
      borderRadius: BorderRadius.circular(24),
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          border: Border.all(color: Colors.grey.shade300, width: 1),
        ),
        child: Image.asset(
          assetName,
          height: 24,
          width: 24,
          errorBuilder: (context, error, stackTrace) {
            // Fallback si l'image n'est pas trouvée
            return const Icon(Icons.login, size: 24);
          },
        ),
      ),
    );
  }
}
