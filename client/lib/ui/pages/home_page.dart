// ignore_for_file: invalid_use_of_protected_member, invalid_use_of_visible_for_testing_member

import 'package:crystal_navigation_bar/crystal_navigation_bar.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:flutter_floating_bottom_bar/flutter_floating_bottom_bar.dart'
    show BottomBar;
import 'package:social_media_app/app/configs/colors.dart';
import 'package:social_media_app/app/configs/theme.dart';
import 'package:social_media_app/ui/bloc/post_cubit.dart';
import 'package:social_media_app/ui/widgets/card_post.dart';

import '../widgets/custom_app_bar.dart';

class HomePage extends StatelessWidget {
  const HomePage({Key? key}) : super(key: key);
  static double iconNormal = 40;
  static ValueNotifier<int> _currentIndex = ValueNotifier(0);
  static ValueNotifier<bool> _showAddMenu = ValueNotifier(false);
  @override
  Widget build(BuildContext context) {
    SystemChrome.setSystemUIOverlayStyle(
      const SystemUiOverlayStyle(
        statusBarColor: Colors.transparent,
        statusBarIconBrightness: Brightness.dark,
      ),
    );
    return Scaffold(
      appBar: _buildCustomAppBar(context),
      extendBodyBehindAppBar: true,
      // extendBody: true,
      body: BottomBar(
        fit: StackFit.expand,
        hideOnScroll: true,
        // start: 0,
        // end: 0,
        offset: 0,
        // reverse: false,
        // respectSafeArea: true,
        iconHeight: 54,
        iconWidth: 54,
        icon: (width, height) => SizedBox(
          child: FittedBox(
            child: Icon(
              Icons.swipe_up_alt_sharp,
              color: AppColors.blackColor.withValues(alpha: 0.5),
              size: width,
            ),
          ),
          width: double.infinity,
          height: height,
        ),
        width: double.infinity,
        barColor: Color(0x00000000),
        body: (context, controller) {
          return Stack(
            alignment: Alignment.bottomCenter,
            children: [
              SingleChildScrollView(
                controller: controller,
                child: Padding(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 16, vertical: 30),
                  child: Column(
                    children: [
                      BlocProvider(
                        create: (context) => PostCubit()..getPosts(),
                        child: BlocBuilder<PostCubit, PostState>(
                          builder: (context, state) {
                            if (state is PostError) {
                              return Center(child: Text(state.message));
                            } else if (state is PostLoaded) {
                              return Padding(
                                padding: const EdgeInsets.only(
                                    bottom: 45.0, top: 54 - 16),
                                child: Column(
                                  children: state.posts
                                      .map((post) => GestureDetector(
                                            child: CardPost(post: post),
                                          ))
                                      .toList(),
                                ),
                              );
                            } else {
                              return const Center(
                                  child: CircularProgressIndicator());
                            }
                          },
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              ValueListenableBuilder<bool>(
                  valueListenable: _showAddMenu,
                  builder: (context, visible, child) {
                    return Positioned.fill(
                      child: Visibility(
                        visible: visible,
                        child: Listener(
                          behavior: HitTestBehavior.translucent,
                          onPointerUp: (event) {
                            _showAddMenu.value = !_showAddMenu.value;
                            _currentIndex.notifyListeners();
                          },
                          child: ExcludeSemantics(
                            child: IgnorePointer(
                              child: AnimatedOpacity(
                                duration: const Duration(milliseconds: 200),
                                opacity: visible ? 1 : 0,
                                child: ColoredBox(
                                    color: AppColors.blackColor
                                        .withValues(alpha: .5)),
                              ),
                            ),
                          ),
                        ),
                      ),
                    );
                  }),
              // ADD ACTIONS
              ValueListenableBuilder<bool>(
                  valueListenable: _showAddMenu,
                  builder: (context, visible, child) {
                    return Positioned(
                      bottom: 85,
                      left: 0,
                      right: 0,
                      child: Listener(
                        behavior: HitTestBehavior.translucent,
                        onPointerUp: (event) {
                          if (_showAddMenu.value) {
                            _showAddMenu.value = false;
                            _currentIndex.notifyListeners();
                          }
                        },
                        child: IgnorePointer(
                          ignoring: !visible,
                          child: AnimatedContainer(
                            duration: const Duration(milliseconds: 200),
                            height: visible ? 300 : 0,
                            margin: EdgeInsets.all(26),
                            width: double.infinity,
                            decoration: BoxDecoration(
                              color: AppColors.whiteColor.withValues(alpha: .5),
                              borderRadius: BorderRadius.circular(30),
                            ),
                            child: GestureDetector(
                              onTap: () {
                                print("TAP ADD");
                              },
                            ),
                          ),
                        ),
                      ),
                    );
                  }),
              _buildBackgroundGradient(),
            ],
          );
        },
        child: _buildBottomNavBar(),
      ),
    );
  }

  Widget _buildBottomNavBar() {
    return ValueListenableBuilder<int>(
        valueListenable: _currentIndex,
        builder: (context, value, child) {
          return _showAddMenu.value
              ? SizedBox(
                  height: 105,
                  child: Center(
                    child: SizedBox(
                      width: 50,
                      height: 50,
                      child: IconButton.filledTonal(
                          color: AppColors.purpleColor,
                          onPressed: () {
                            _showAddMenu.value = !_showAddMenu.value;
                            _currentIndex.notifyListeners();
                          },
                          icon: Icon(Icons.close)),
                    ),
                  ),
                )
              : CrystalNavigationBar(
                  currentIndex: value,
                  indicatorColor: AppColors.whiteColor,
                  unselectedItemColor:
                      AppColors.whiteColor.withValues(alpha: 1),
                  backgroundColor: AppColors.blackColor.withValues(alpha: 0.3),
                  outlineBorderColor:
                      AppColors.purpleColor.withValues(alpha: 0.1),
                  borderWidth: 2,
                  selectedItemColor:
                      AppColors.purpleColor.withValues(alpha: 0.9),
                  enableFloatingNavBar: true,
                  boxShadow: [
                    BoxShadow(
                        color: AppColors.greyColor.withValues(alpha: 0.2),
                        spreadRadius: 0,
                        blurRadius: 0,
                        offset: Offset(0, 0))
                  ],
                  onTap: (i) {
                    if (i != 2) {
                      _currentIndex.value = i;
                    } else {
                      _showAddMenu.value = !_showAddMenu.value;
                      _currentIndex.notifyListeners();
                    }
                  },
                  items: [
                    /// Home
                    CrystalNavigationBarItem(
                      icon: Icons.home,
                      unselectedIcon: Icons.home,
                      badge: Badge(
                        label: Text(
                          "9+",
                          style: TextStyle(color: Colors.white),
                        ),
                      ),
                    ),

                    /// Favourite
                    CrystalNavigationBarItem(
                      icon: Icons.favorite,
                      unselectedIcon: Icons.favorite,
                      selectedColor: Colors.red,
                    ),

                    /// Add
                    CrystalNavigationBarItem(
                      icon: Icons.add,
                      // unselectedIcon: Icons.add,
                    ),

                    /// Search
                    CrystalNavigationBarItem(
                      icon: Icons.search,
                      unselectedIcon: Icons.search,
                    ),

                    /// Profile
                    CrystalNavigationBarItem(
                      icon: Icons.verified_user,
                      unselectedIcon: Icons.verified_user_outlined,
                    ),
                  ],
                );
        });
  }

  Container __buildBottomNavBar() {
    return Container(
      width: double.infinity,
      height: 110,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      margin: const EdgeInsets.only(right: 24, left: 24, bottom: 16),
      decoration: BoxDecoration(
        color: AppColors.whiteColor,
        borderRadius: BorderRadius.circular(30),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          _buildItemBottomNavBar("assets/images/ic_home.png", "Home", true),
          _buildItemBottomNavBar(
              "assets/images/ic_discorvery.png", "Discover", false),
          _buildItemBottomNavBar("assets/images/ic_inbox.png", "Inbox", false),
          _buildItemBottomNavBar(
              "assets/images/ic_profile.png", "Profile", false),
        ],
      ),
    );
  }

  _buildItemBottomNavBar(String icon, String title, bool selected) {
    return Container(
      width: 70,
      height: 70,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(24),
        color: selected ? AppColors.whiteColor : Colors.transparent,
        boxShadow: [
          if (selected)
            BoxShadow(
              color: AppColors.blackColor.withValues(alpha: 0.1),
              blurRadius: 35,
              offset: const Offset(0, 10),
            ),
        ],
      ),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Image.asset(
            icon,
            width: 24,
            height: 24,
            color: selected ? AppColors.purpleColor : AppColors.blackColor,
          ),
          const SizedBox(height: 4),
          Text(
            title,
            style: AppTheme.blackTextStyle.copyWith(
              fontWeight: AppTheme.bold,
              fontSize: 12,
              color: selected ? AppColors.purpleColor : AppColors.blackColor,
            ),
          ),
        ],
      ),
    );
  }

  _buildBackgroundGradient() => Container(
        width: double.infinity,
        height: 150,
        decoration: BoxDecoration(
          gradient: LinearGradient(colors: [
            AppColors.whiteColor.withValues(alpha: 0),
            AppColors.whiteColor.withValues(alpha: 0.8),
          ], begin: Alignment.topCenter, end: Alignment.bottomCenter),
        ),
      );

  AppBar _buildCustomAppBar(BuildContext context) {
    return AppBar(
      backgroundColor: AppColors.whiteColor.withValues(alpha: 0.7),
      bottomOpacity: 0.5,
      // elevation: 0,
      // shadowColor: Colors.transparent,
      // animateColor: false,
      // centerTitle: true,
      // primary: false,
      // surfaceTintColor: Colors.transparent,
      // forceMaterialTransparency: true,
      leading: Container(
        width: 40,
        height: 40,
        margin: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
        decoration: BoxDecoration(
            // boxShadow: [
            //   BoxShadow(
            //     color: AppColors.blackColor.withValues(alpha: 0.2),
            //     blurRadius: 35,
            //     offset: const Offset(0, 10),
            //   ),
            // ],
            ),
        child: Image.asset(
          'assets/images/ic_logo.png',
          width: 40,
          height: 40,
        ),
      ),
      actions: [
        Image.asset("assets/images/ic_search.png", width: 24, height: 24),
        const SizedBox(width: 12),
        Image.asset("assets/images/ic_notification.png", width: 24, height: 24),
        const SizedBox(width: 12),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 6),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(35),
            color: AppColors.backgroundColor,
          ),
          child: Row(
            children: [
              Container(
                width: 32,
                height: 32,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  border: Border.all(
                    color: AppColors.whiteColor,
                    width: 1,
                  ),
                  boxShadow: [
                    BoxShadow(
                      color: AppColors.blackColor.withValues(alpha: 0.1),
                      blurRadius: 10,
                      offset: const Offset(0, 10),
                    ),
                  ],
                  image: const DecorationImage(
                    fit: BoxFit.cover,
                    image: AssetImage(
                      "assets/images/img_profile.jpeg",
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 6),
              Text(
                "Sajon.co",
                style: AppTheme.blackTextStyle
                    .copyWith(fontWeight: AppTheme.bold, fontSize: 12),
              ),
              const SizedBox(width: 2),
              Image.asset(
                "assets/images/ic_checklist.png",
                width: 16,
              ),
              const SizedBox(width: 4),
            ],
          ),
        ),
        const SizedBox(width: 8),
      ],
    );
  }
}

class BorderPainter extends CustomPainter {
  final Path path;
  final Color borderColor;
  final double borderWidth;
  BorderPainter(
      {required this.path,
      this.borderColor = Colors.black,
      this.borderWidth = 2.0});

  @override
  void paint(Canvas canvas, Size size) {
    Paint paint = Paint()
      ..color = borderColor
      ..style =
          PaintingStyle.stroke // Key property for drawing an outline [1.3]
      ..strokeWidth = borderWidth;
    canvas.drawPath(path, paint);
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldPainter) => true;
}
